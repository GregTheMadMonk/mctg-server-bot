// bot.go
// Main bot package file
package bot

import (
    "bytes"
    "errors"
    "fmt"
    "golang.org/x/image/webp"
    "image"
    "image/gif"
    "image/jpeg"
    "image/png"
    "log"
    "regexp"
    "sort"
    "strings"
    "sync"

    "github.com/gregthemadmonk/mctg-server-bot/tg_api"
)

// Bot config
type Config struct {
    // Telegram bot API token
    ApiToken string `json:"api_token"`
    // Telegram channel ID for the bot to live in
    ChatId int `json:"chat_id"`
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

func (self *bot) FileUri(filePath string) string {
    return fmt.Sprintf(
        "%s%s/%s", tg_api.API_FILE_BASE, self.config.ApiToken, filePath,
    )
} // <-- bot::FileUri(filePath)

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

    params := Params{0}

    updateMessage := func(message *tg_api.Message) any {
        if message.Chat.Id != self.config.ChatId {
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

        if strings.HasPrefix(message.Text, "/iamthe") {
            argv := strings.Split(message.Text, " ")
            if len(argv) != 2 {
                return OutputEventUserError{
                    Message: "Usage: /iamthe <minecraft_nickname>",
                }
            }

            return OutputEventBindUser{
                TelegramName:  message.From.Username,
                MinecraftName: argv[1],
            }
        }

        if admin && message.Text[0] == '/' {
            return OutputEventCommand{message.Text}
        }

        messageEvent := OutputEventMessage{Username: message.From.Username}

        if len(message.Text) > 0 {
            messageEvent.AddText(message.Text)
        }

        if len(message.Caption) > 0 {
            messageEvent.AddText(message.Caption)
        }

        if message.Sticker != nil {
            im, err := self.get_image(message.Sticker.FileId)
            if err != nil {
                messageEvent.AddImage(im)
            } else {
                messageEvent.AddText("[unsupported sticker]") // Normally this needs to be a separate type...
            }
        }

        if len(message.Photo) > 0 {
            // Get version of photo with higher resolution
            sort.Slice(message.Photo, func(i, j int) bool {
                return message.Photo[i].Width < message.Photo[j].Width
            })

            im, _ := self.get_image(message.Photo[0].FileId)
            messageEvent.AddImage(im)
        }

        if len(messageEvent.GetMessage()) == 0 {
            return nil
        }

        return messageEvent
    } // <-- updateMessage(message)

    exch_f := tg_api.ExchangeIntoWith[[]tg_api.Update, Params]
    for {
        if self.running != BS_RUNNING {
            break
        }

        if res, err := exch_f(self.Uri("getUpdates"), params); err == nil {
            if !res.Ok {
                self.out <- OutputEventAPIError{res}
                continue
            }

            for _, update := range res.Result {
                if params.Offset < update.UpdateId+1 {
                    params.Offset = update.UpdateId + 1
                }

                if update.Message != nil {
                    if e := updateMessage(update.Message); e != nil {
                        self.out <- e
                    }
                }

                if update.EditedMessage != nil && len(update.EditedMessage.Text) != 0 {
                    if update.EditedMessage.Chat.Id == self.config.ChatId {
                        self.out <- OutputEventEditMessage{
                            Username: update.EditedMessage.From.Username,
                            Message:  update.EditedMessage.Text,
                        }
                    }
                }
            }
        } else {
            self.out <- OutputEventRequestError{err}
        }
    }
    self.wg.Done()
    log.Println("Exit bot.bot::handle_updates()")
} // <-- bot::handle_updates()

func (self *bot) decode_image(content []byte, extension string) (image.Image, error) {
    contentBuffer := bytes.NewBuffer(content)

    var im image.Image
    var err error

    switch extension {
    case ".webp":
        im, err = webp.Decode(contentBuffer)
        break
    case ".jpg":
        im, err = jpeg.Decode(contentBuffer)
        break
    case ".png":
        im, err = png.Decode(contentBuffer)
        break
    case ".gif":
        im, err = gif.Decode(contentBuffer)
        break
    default:
        err = errors.New("unsupported image format")
        break
    }

    if err != nil {
        log.Println("Failed to decode image:", err)
        return nil, err
    }

    return im, nil
}

func (self *bot) get_image(fileId string) (image.Image, error) {
    p := tg_api.GetFile{FileId: fileId}

    log.Println("Preparing file for downloading:", fileId)
    f, err := tg_api.ExchangeIntoWith[tg_api.File, tg_api.GetFile](self.Uri("getFile"), p)
    if err != nil {
        log.Println("Failed to get file info", fileId)
        return nil, err
    }

    file := f.Result

    log.Println("Get file content:", file.FilePath)
    content, err := tg_api.Exchange(self.FileUri(file.FilePath))
    if err != nil {
        log.Println("Failed to get file content", fileId)
        return nil, err
    }

    extension := regexp.MustCompile("\\.\\w+").FindString(file.FilePath)

    im, err := self.decode_image(*content, extension)

    if err != nil {
        log.Println("Failed to decode image", fileId)
        return nil, err
    }

    return im, nil
} // <-- bot::get_file(fileId)

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
