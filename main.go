package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/exec"
    "regexp"
    "strconv"
    "strings"
    "time"
) // <-- import

// The name of an environment variable that holds the bot token
const TOKEN_ENV = "SERVERBOT_TOKEN"
// The name of en environment variable that holds the chat ID for bot
const CHAT_ENV  = "SERVERBOT_CHAT"
// The name of an environment variable that holds the Telegram admin username
const ADMIN_ENV string = "SERVERBOT_ADMIN"

// To take address of it
const MARKDOWN = "MarkdownV2"

// Max log lines to store
const LOG_LINES = 15

// Telegram bot API URL
const TG_API = "https://api.telegram.org/bot"

// Full bot URL with token
var bot_url string
// Bot chat id
var bot_chat_id int
// Bot tg admin username
var bot_admin_user string

// Log message ID
var log_message_id *int = nil

// Server logs
var logs []string

// Last log message (can't be the same or Tg will throw error)
var last_log_message string

// Current online players
var players_online []string

// Telgram API returns all results with optional error info
type ExchangeResult[T any] struct {
    Ok          bool   `json:"ok"`
    ErrorCode   int    `json:"error_code"`
    Description string `json:"description"`
    Result      T      `json:"result"`
} // <-- struct ExchangeResult[T]

type GetMe struct {
    Id                      int
    IsBot                   bool   `json:"is_bot"`
    FirstName               string `json:"first_name"`
    Username                string `json:"username"`
    CanJoinGroups           bool   `json:"can_join_groups"`
    CanReadAllGroupMessages bool   `json:"can_read_all_group_messages"`
    SupportsInlineQueries   bool   `json:"supports_inline_queries"`
    CanConnectToBusiness    bool   `json:"can_connect_to_business"`
    HasMainWebApp           bool   `json:"has_main_web_app"`
} // <-- struct GetMe

type User struct {
    Username string `json:"username"`
} // <-- struct User

type Message struct {
    MessageId int    `json:"message_id"`
    From      User   `json:"from"`
    Text      string `json:"text"`
} // <-- struct Message

type Update struct {
    UpdateId          int      `json:"update_id"`
    Message           *Message `json:"message"`
    EditedMessage     *Message `json:"edited_message"`
    ChannelPost       *Message `json:"channel_post"`
    EditedChannelPost *Message `json:"edited_channel_post"`
    // TODO: Implement other fields as needed
} // <-- struct Update

type SendMessage struct {
    ChatId    int    `json:"chat_id"`
    Text      string `json:"text"`
    ParseMode string `json:"parse_mode,omitempty"`
} // <-- struct SendMessage

type EditMessageText struct {
    ChatId    int    `json:"chat_id"`
    MessageId int    `json:"message_id"`
    Text      string `json:"text"`
    ParseMode string `json:"parse_mode,omitempty"`
} // <-- struct EditMessageText

func push_player(username string) {
    found := false
    for _, player := range players_online {
        found = found || (player == username)
    }
    if !found {
        players_online = append(players_online, username)
    }
} // <-- push_player(username)

func pop_player(username string) {
    for i, player := range players_online {
        if player == username {
            players_online = append(players_online[:i], players_online[i+1:]...)
            return
        }
    }
} // <-- pop_player(username)

func exchange(endpoint string) (*[]byte, error) {
    if res, e := http.Get(fmt.Sprintf("%s%s", bot_url, endpoint)); e == nil {
        if body, err_b := io.ReadAll(res.Body); err_b == nil {
            return &body, nil
        } else {
            return nil, err_b
        }
    } else {
        return nil, e
    }
} // <-- exchange(endpoin)

func exchange_with[P any](endpoint string, params P) (*[]byte, error) {
    uri := fmt.Sprintf("%s%s", bot_url, endpoint)
    if body, js_e := json.Marshal(params); js_e == nil {
        // log.Println("Sending", string(body))
        rqb := bytes.NewBuffer(body)
        if res, e := http.Post(uri, "application/json", rqb); e == nil {
            if resp_body, err_b := io.ReadAll(res.Body); err_b == nil {
                return &resp_body, nil
            } else {
                return nil, err_b
            }
        } else {
            return nil, e
        }
    } else {
        return nil, js_e
    }
} // <-- exchange_with(endpoint, params)

func exchange_into[T any](endpoint string) (*ExchangeResult[T], error) {
    if res, err := exchange(endpoint); err == nil {
        var ret ExchangeResult[T]
        // log.Println("Deserializing ", string(*res))
        if js_err := json.Unmarshal(*res, &ret); js_err != nil {
            return nil, js_err
        }
        return &ret, nil
    } else {
        return nil, err
    }
} // <-- exchange_into[T](endpoint)

func exchange_into_with[T any, P any](endpoint string, params P) (*ExchangeResult[T], error) {
    if res, err := exchange_with(endpoint, params); err == nil {
        var ret ExchangeResult[T]
        // log.Println("Deserializing ", string(*res))
        if js_err := json.Unmarshal(*res, &ret); js_err != nil {
            return nil, js_err
        }
        return &ret, nil
    } else {
        return nil, err
    }
} // <-- exchange_into[T](endpoint)

func send_message(message string, markdown bool) *Message {
    p := SendMessage{
        ChatId:    bot_chat_id,
        Text:      message,
    }

    if markdown {
        p.ParseMode = MARKDOWN
    } else {
        p.ParseMode = ""
    }

    if res, err := exchange_into_with[Message]("sendMessage", p); err == nil {
        if !(*res).Ok {
            log.Fatalln(
                "Failed to send message: API error",
                (*res).ErrorCode,
                (*res).Description,
                " while sending\"",
                message,
                "\"",
            )
            return nil
        }
        return &(*res).Result
    } else {
        log.Fatalln("Failed to send message: ", err)
        return nil
    }
} // <-- send_message(message)

func edit_message(message_id int, message string, markdown bool) {
    p := EditMessageText{
        ChatId:    bot_chat_id,
        MessageId: message_id,
        Text:      message,
    }

    if markdown {
        p.ParseMode = MARKDOWN
    } else {
        p.ParseMode = ""
    }

    if r, e := exchange_into_with[Message]("editMessageText", p); e == nil {
        if !(*r).Ok {
            log.Fatalln(
                "Failed to edit message: API error",
                (*r).ErrorCode,
                (*r).Description,
                " while sending\"",
                message,
                "\"",
            )
        }
    } else {
        log.Fatalln("Failed to edit message:", e)
    }
} // <-- edit_message(message_id, message)

func read_pipe_loop(pipe_in io.ReadCloser) {
    reader := bufio.NewReader(pipe_in)

    // Not secure because I don't care about security 😎
    message_r := regexp.MustCompile(
        `\[minecraft/MinecraftServer\]: \[Not Secure\] \<(?P<user>.*)\> (?P<msg>.*)`,
    )
    achievement_r := regexp.MustCompile(
        `\[minecraft/MinecraftServer\]: (?P<user>.*) has made the advancement \[(?P<adv>.*)\]`,
    )
    joined_r := regexp.MustCompile(
        `\[minecraft/MinecraftServer\]: (?P<user>.*) joined the game`,
    )
    left_r := regexp.MustCompile(
        `\[minecraft/MinecraftServer\]: (?P<user>.*) left the game`,
    )

    for {
        if str, err := reader.ReadString('\n'); err == nil {
            log.Println("SERVER", str)

            logs = append(logs, str)
            if len(logs) > LOG_LINES {
                logs = logs[1:]
            }

            if sm := message_r.FindStringSubmatch(str); sm != nil {
                send_message(
                    fmt.Sprintf(
                        "%s: %s", sm[1], sm[2],
                    ), false,
                )
            } else if sm := achievement_r.FindStringSubmatch(str); sm != nil {
                send_message(
                    fmt.Sprintf(
                        "%s has made the advancement %s", sm[1], sm[2],
                    ), false,
                )
            } else if sm := joined_r.FindStringSubmatch(str); sm != nil {
                send_message(fmt.Sprintf("%s joined the game", sm[1]), false)
                push_player(sm[1])
            } else if sm := left_r.FindStringSubmatch(str); sm != nil {
                send_message(fmt.Sprintf("%s left the game", sm[1]), false)
                pop_player(sm[1])
            } else if strings.Contains(str, "[minecraft/DedicatedServer]: Done ") {
                send_message("Server started!", false)
            }
        } else {
            log.Fatalln("Failed reading stdin")
        }
    }
} // <-- read_pipe_loop(pipe_in, pipe_out)

func telegram_api_loop(pipe_out io.WriteCloser) {
    var params struct {
        Offset int `json:"offset"`
    }

    params.Offset = 0
    for {
        if res, err := exchange_into_with[[]Update]("getUpdates", params); err == nil {
            if !(*res).Ok {
                log.Fatalln("API error: ", (*res).ErrorCode, (*res).Description)
            }

            for _, update := range (*res).Result {
                // Bump the update offset to prevent us from reading the same
                // thing over and over again. This will also notify Telegram
                // that we've received the updates next time we make a request
                if params.Offset < update.UpdateId + 1 {
                    params.Offset = update.UpdateId + 1
                }

                // Got a message
                if update.Message != nil && len(update.Message.Text) != 0 {
                    if update.Message.Text == "/players" {
                        send_message(
                            fmt.Sprintf(
                                "%d active players:\n%v\n",
                                len(players_online),
                                players_online,
                            ), false,
                        )
                    } else if update.Message.Text[0] == '/' && update.Message.From.Username == bot_admin_user {
                        fmt.Fprintf(pipe_out, "%s\n", update.Message.Text)
                    } else {
                        for _, line := range strings.Split(update.Message.Text, "\n") {
                            fmt.Fprintf(
                                pipe_out,
                                "/say §6@%s§f: %s\n",
                                update.Message.From.Username,
                                line,
                            )
                        }
                    }
                }
                if update.EditedMessage != nil {
                    for _, line := range strings.Split(update.EditedMessage.Text, "\n") {
                        fmt.Fprintf(
                            pipe_out,
                            "/say §6@%s§8 corrects§f: %s\n",
                            update.EditedMessage.From.Username,
                            line,
                        )
                    }
                }
            }
        } else {
            log.Fatalln("Cannot get updates for the bot")
        }
    }
} // <-- telegram_api_loop()

func send_logs() {
    escape_chars := []string{
        "`",
        `>`,
        `#`,
        `|`,
        `{`,
        `}`,
        `!`,
        `\`,
    }

    for {
        log_msg := ""
        for _, line := range logs {
            log_msg = log_msg + "\n" + line
        }
        for _, char := range escape_chars {
            log_msg = strings.ReplaceAll(log_msg, char, `\` + char)
        }
        log_msg = fmt.Sprintf("Server logs:\n```\n%s\n```\n", log_msg)

        if log_message_id == nil {
            log_message_id = &(*send_message(log_msg, true)).MessageId
        } else {
            if last_log_message != log_msg {
                edit_message(*log_message_id, log_msg, true)
            }
        }

        last_log_message = log_msg

        time.Sleep(5 * time.Second)
    }
}

func run_server() {
    cmd := exec.Command(os.Args[1], os.Args[2:]...)
    pipe_in, in_err := cmd.StdinPipe()
    if in_err != nil {
        log.Fatalln("Cannot open pipe for child stdin:", in_err)
    }
    pipe_out, out_err := cmd.StdoutPipe()
    if out_err != nil {
        log.Fatalln("Cannot open pipe for child stdout:", out_err)
    }

    if err := cmd.Start(); err != nil {
        log.Fatalln("Could not start subprocess", err)
    }

    // Out and in are swapped: child's input is our output and vice-versa
    go read_pipe_loop(pipe_out)
    go telegram_api_loop(pipe_in)
    go send_logs()

    cmd.Wait()
    log.Println("Server exited, shutting down the bot")
    os.Exit(0)
} // <-- run_server()

func main() {
    if len(os.Args) < 2 {
        log.Fatalln("Please provide the server command line in arguments")
    }

    if token, found := os.LookupEnv(TOKEN_ENV); found {
        log.Println("Token found in", TOKEN_ENV)
        bot_url = fmt.Sprintf("%s%s/", TG_API, token)
    } else {
        log.Fatalln("Token not provided!")
    }

    if chat, found := os.LookupEnv(CHAT_ENV); found {
        if bci, err := strconv.Atoi(chat); err == nil {
            bot_chat_id = bci
        } else {
            log.Fatalln(CHAT_ENV, "must be an integer")
        }
    } else {
        log.Fatalln("Telegram chat ID not provided in", CHAT_ENV)
    }

    if admin_user, found := os.LookupEnv(ADMIN_ENV); found {
        bot_admin_user = admin_user
    } else {
        log.Println("Running without an admin username :(")
    }

    log.Println("Checking Telegram bot API accessibility...")
    if res, err := exchange_into[GetMe]("getMe"); err == nil {
        if !(*res).Ok {
            log.Fatalln(
                "Telegram API reported error:",
                (*res).ErrorCode,
                (*res).Description,
            )
        }
        log.Printf(
            "Running as @%s (%s)\n",
            (*res).Result.Username,
            (*res).Result.FirstName,
        )
    } else {
        log.Fatalln("Cannot fetch bot API getMe:", err)
    }

    run_server()
} // <-- main()
