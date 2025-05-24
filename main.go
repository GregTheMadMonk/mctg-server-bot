package main

import (
    "encoding/json"
    "fmt"
    "github.com/gregthemadmonk/mctg-server-bot/bot"
    "github.com/gregthemadmonk/mctg-server-bot/server"
    "log"
    "os"
) // <-- import

type Config struct {
    Bot    bot.Config    `json:"bot"`
    Server server.Config `json:"server"`
} // <-- type Config struct

func main() {
    log.Println("Loading config...")
    var config Config

    if cfg, err := os.ReadFile("mctg-bot-config.json"); err == nil {
        if json_err := json.Unmarshal(cfg, &config); json_err != nil {
            log.Fatalln("Could not parse mctg-bot-config.json:", json_err)
        }
    } else {
        log.Fatalln("Could not load mctg-bot-config.json:", err)
    }

    log.Println("Initializing the Telegram bot...")
    thebot, bot_err := bot.MakeBot(config.Bot)
    if bot_err != nil {
        log.Fatalln("Could not initialize bot:", bot_err)
    }
    srv := server.MakeHandle(config.Server)
    if srv_err := srv.Start(); srv_err != nil {
        log.Fatalln("Could not initialize server:", srv_err)
    }

    tg_msg_name := func(usr string) string {
        tg_name := srv.ReverseRename(usr)
        if tg_name == usr {
            return usr
        }
        return fmt.Sprintf("%s (%s)", usr, tg_name)
    } // <-- tg_msg_name(usr)

    // Main loop
    // The .In() channels could've been replaced with calling methods but
    // the code was easier to design around separate goroutines handling
    // input and output
    stopping := false
    thebot.Start()
    for {
        if stopping && !srv.IsRunning() && !thebot.IsRunning() {
            break
        }

        select {
        case srv_out := <-srv.Out():
            switch event := srv_out.(type) {
            case server.OutputEventExit:
                log.Println("Server exited, got code", event.ExitCode)
                thebot.In() <- bot.InputEventSendMessage{
                    Message: fmt.Sprintf(
                        "Server shut down with exit code %d",
                        event.ExitCode,
                    ),
                }

                stopping = true
                if srv.TryRestart {
                    thebot.In() <- bot.InputEventSendMessage{
                        Message: "Restarting...",
                    }
                    if srv_err := srv.Start(); srv_err != nil {
                        thebot.In() <- bot.InputEventSendMessage{
                            Message: "Oof",
                        }
                        log.Println("Tried to restart the server, but couldnt")
                        log.Println(srv_err)
                    } else {
                        stopping = false
                    }
                }

                if stopping {
                    thebot.Stop()
                }
            case server.OutputEventPlayerJoined:
                thebot.In() <- bot.InputEventSendMessage{
                    Message: fmt.Sprintf(
                        "%s joined the game",
                        tg_msg_name(event.Username),
                    ),
                }
            case server.OutputEventPlayerLeft:
                thebot.In() <- bot.InputEventSendMessage{
                    Message: fmt.Sprintf(
                        "%s left the game",
                        tg_msg_name(event.Username),
                    ),
                }
            case server.OutputEventPlayerAchievement:
                thebot.In() <- bot.InputEventSendMessage{
                    Message: fmt.Sprintf(
                        "%s has achieved: %s",
                        tg_msg_name(event.Username),
                        event.Achievement,
                    ),
                }
            case server.OutputEventServerLoaded:
                thebot.In() <- bot.InputEventSendMessage{
                    Message: "Server successfully started",
                }
            case server.OutputEventListPlayers:
                msg := fmt.Sprintf("%d players:\n", len(event.PlayersOnline))
                for _, player := range event.PlayersOnline {
                    msg += fmt.Sprintf("* %s\n", tg_msg_name(player))
                }
                thebot.In() <- bot.InputEventSendMessage{Message: msg}
            case server.OutputEventLog:
                log.Println(event.Message)
            case server.OutputEventMessage:
                thebot.In() <- bot.InputEventSendMessage{
                    Message: fmt.Sprintf(
                        "%s: %s",
                        tg_msg_name(event.Username),
                        event.Message,
                    ),
                }

                if event.Tellraw {
                    // Server has the mod installed, this message should be
                    // re-relayed to the server too
                    srv.In() <- *server.InputEventChat{
                        Telegram: false,
                        Username: event.Username,
                    }.Build().AddText(event.Message)
                }
            case server.OutputEventPlayerDeath:
                thebot.In() <- bot.InputEventSendMessage{
                    Message: fmt.Sprintf(
                        "%s: %s\nYikes...\n",
                        tg_msg_name(event.Username),
                        event.Message,
                    ),
                }
            case server.OutputEventError:
                log.Println("Server error:", event.Error)
                if event.Error.Error() == "User error" {
                    thebot.In() <- bot.InputEventSendMessage{
                        Message: "User error",
                    }
                }
            default:
                log.Println("Unknown event sent by server:", srv_out)
            }
        case bot_out := <-thebot.Out():
            switch event := bot_out.(type) {
            case bot.OutputEventMessage:
                chatEvent, err := server.InputEventChat{
                    Telegram: true,
                    Username: event.Username,
                }.Build().AddMessageParts(event.GetMessage())

                if err != nil {
                    log.Println(err)
                } else {
                    srv.In() <- *chatEvent
                }
            case bot.OutputEventEditMessage:
                srv.In() <- server.InputEventEditChat{
                    Username: event.Username,
                    Message:  event.Message,
                }
            case bot.OutputEventCommand:
                srv.In() <- server.InputEventCommand{
                    Command: event.Command,
                }

            case bot.OutputEventListPlayers:
                srv.In() <- server.InputEventListPlayers{}
            case bot.OutputEventKillServer:
                srv.In() <- server.InputEventKillServer{}
            case bot.OutputEventBindUser:
                srv.In() <- server.InputEventBindRename{
                    Username:    event.TelegramName,
                    DisplayName: event.MinecraftName,
                }
            case bot.OutputEventUserError:
                thebot.In() <- bot.InputEventSendMessage{
                    Message: event.Message,
                }
            case bot.OutputEventAPIError:
                log.Println("Telegram API error:", event.Error)
            default:
                log.Println("Unknown event sent by bot:", bot_out)
            }
        }
    }
} // <-- main()
