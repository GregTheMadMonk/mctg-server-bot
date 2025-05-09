// tg_api_types.go
// Types for interacting with Telegram API. As complete as I needed them to be
package tg_api

// Telgram API returns all results with optional error info
type ExchangeResult[T any] struct {
    Ok          bool   `json:"ok"`
    ErrorCode   int    `json:"error_code"`
    Description string `json:"description"`
    Result      T      `json:"result"`
} // <-- struct ExchangeResult[T]

type GetMe struct {
    Id                      int    `json:"id"`
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

type Chat struct {
    Id int `json:"id"`
} // <-- struct Chat

type Message struct {
    MessageId int    `json:"message_id"`
    From      User   `json:"from"`
    Text      string `json:"text"`
    Chat      Chat   `json:"chat"`
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
