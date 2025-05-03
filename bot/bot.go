// bot.go
// Main bot package file
package bot

import (
    "fmt"
    "log"
    "sync"

    "github.com/gregthemadmonk/mctg-server-bot/tg_api"
)

// Bot config
type Config struct {
    // Telegram bot API token
    ApiToken      string `json:"api_token"`
    // Telegram channel ID for the bot to live in
    ChatId        int    `json:"chat_id"`
    // Username of a Telegram user who can issue slash-commands directly to
    // the server
    AdminUsername string `json:"admin_username,omitempty"`
} // <-- struct Config

const (
    BS_RUNNING  = iota
    BS_STOPPING = iota
    BS_STOPPED  = iota
)

// The bot state
type bot struct {
    config  Config
    running uint
    wg      sync.WaitGroup
    out     chan any
    in      chan any
} // <-- struct bot

// Create bot from the config
func MakeBot(
    bot_cfg Config,
) (*bot, error) {
    ret := bot{
        config:  bot_cfg,
        running: BS_STOPPED,
        out:     make(chan any),
        in:      make(chan any),
    }

    log.Println("Checking Telegram bot API accessibility...")
    url := ret.Uri("getMe")
    if res, err := tg_api.ExchangeInto[tg_api.GetMe](url); err == nil {
        if !res.Ok {
            return nil, res
        }
        log.Printf(
            "Running as @%s (%s)\n",
            res.Result.Username,
            res.Result.FirstName,
        )
    } else {
        return nil, err
    }

    return &ret, nil
} // <-- MakeBot(cfg)

// Get bot's output event channel
func (self *bot) Out() <-chan any {
    return self.out
} // <-- bot::Out()

// Get bot's input event channel
func (self *bot) In() chan<- any {
    return self.in
} // <-- bot::In()

func (self *bot) Uri(endpoint string) string {
    return fmt.Sprintf(
        "%s%s/%s", tg_api.API_BASE, self.config.ApiToken, endpoint,
    )
} // <-- bot::Uri(endpoint)

// Send message as a bot. Set `use_md=true` if message contains Markdown
func (self *bot) send_message(
    message string, use_md bool,
) (*tg_api.Message, error) {
    p := tg_api.SendMessage{
        ChatId:    self.config.ChatId,
        Text:      message,
        ParseMode: "",
    }

    if use_md {
        p.ParseMode = tg_api.PM_MARKDOWN
    }

    exch_f := tg_api.ExchangeIntoWith[tg_api.Message, tg_api.SendMessage]
    if res, err := exch_f(self.Uri("sendMessage"), p); err == nil {
        if !res.Ok {
            return nil, res
        }
        return &res.Result, nil
    } else {
        return nil, err
    }
} // <-- bot::send_message(message, use_md)

// Make a bot edit its message. Set `use_md=true` if message contains Markdown
func (self *bot) edit_message(
    message_id int, message string, use_md bool,
) (*tg_api.Message, error) {
    p := tg_api.EditMessageText{
        ChatId:    self.config.ChatId,
        MessageId: message_id,
        Text:      message,
        ParseMode: "",
    }
    
    if use_md {
        p.ParseMode = tg_api.PM_MARKDOWN
    }

    exch_f := tg_api.ExchangeIntoWith[tg_api.Message, tg_api.EditMessageText]
    if res, err := exch_f(self.Uri("editMessageText"), p); err == nil {
        if !res.Ok {
            return nil, res
        }
        return &res.Result, nil
    } else {
        return nil, err
    }
} // <-- bot::edit_message(message_id, message, use_md)

func (self *bot) IsRunning() bool { return self.running != BS_STOPPED }

func (self *bot) handle_updates() {
    type Params struct {
        Offset int `json:"offset"`
    } // <-- var params

    params := Params{ 0 }

    updateMessage := func(message *tg_api.Message) any {
        if len(message.Text) == 0 {
            return nil
        }

        admin := message.From.Username == self.config.AdminUsername

        switch message.Text {
        case "/players":
            return OutputEventListPlayers{}
        case "/kill-server":
            if admin {
                return OutputEventKillServer{}
            }
        }

        if admin && message.Text[0] == '/' {
            return OutputEventCommand{ message.Text }
        }

        return OutputEventMessage{
            Username: message.From.Username,
            Message:  message.Text,
        }
    } // <-- updateMessage(message)

    exch_f := tg_api.ExchangeIntoWith[[]tg_api.Update, Params]
    for {
        if self.running != BS_RUNNING {
            break
        }

        if res, err := exch_f(self.Uri("getUpdates"), params); err == nil {
            if !res.Ok {
                self.out <- OutputEventAPIError{ res }
                continue
            }

            for _, update := range res.Result {
                if params.Offset < update.UpdateId + 1 {
                    params.Offset = update.UpdateId + 1
                }

                if update.Message != nil {
                    if e := updateMessage(update.Message); e != nil {
                        self.out <- e
                    }
                }

                if update.EditedMessage != nil && len(update.EditedMessage.Text) != 0 {
                    self.out <- OutputEventEditMessage{
                        Username: update.EditedMessage.From.Username,
                        Message:  update.EditedMessage.Text,
                    }
                }
            }
        } else {
            self.out <- OutputEventRequestError{ err }
        }
    }
    self.wg.Done()
    log.Println("Exit bot.bot::handle_updates()")
} // <-- bot::handle_updates()

func (self *bot) handle_inputs() {
    handler:
    for {
        ie, open := <-self.in
        if !open {
            continue
        }

        switch event := ie.(type) {
        case input_event_terminate:
            break handler
        case InputEventSendMessage:
            self.send_message(event.Message, false)
        }
    }
    self.wg.Done()
    log.Println("Exit bot.bot::handle_inputs()")
} // <-- bot::handle_input()

func (self *bot) Start() {
    if self.running != BS_STOPPED {
        log.Println("Trying to start the bot twice!")
        return
    }

    self.running = BS_RUNNING
    self.wg.Add(2)
    go self.handle_updates()
    go self.handle_inputs()
} // <-- bot::Start()

func (self *bot) Stop() {
    if self.running != BS_RUNNING {
        log.Println("Trying to stop the bot twice!")
        return
    }

    self.running = BS_STOPPING
    self.in <- input_event_terminate{}
    self.wg.Wait()
    self.running = BS_STOPPED
} // <-- bot::Stop()
