package bot

import "image"

type OutputEventMessage struct {
    Username string
    message  []any // string or image.Image
} // <-- struct OutputEventMessage

func (self OutputEventMessage) Build() *OutputEventMessage {
    return &self
}

func (self *OutputEventMessage) GetMessage() []any { return self.message }

func (self *OutputEventMessage) AddText(text string) *OutputEventMessage {
    self.message = append(self.message, text)
    return self
}

func (self *OutputEventMessage) AddImage(image image.Image) *OutputEventMessage {
    self.message = append(self.message, image)
    return self
}

type OutputEventEditMessage struct {
    Username string
    Message  string
} // <-- sturct OutputEventEditMessage

type OutputEventCommand struct {
    Command string
} // <-- struct OutputEventCommand

type OutputEventBindUser struct {
    TelegramName  string
    MinecraftName string
} // <-- struct OutputEventBindUser

type OutputEventListPlayers struct{}

type OutputEventKillServer struct{}

type OutputEventUserError struct {
    Message string
} // <-- struct OutputEventUserError

type OutputEventRequestError struct {
    Error error
} // <-- struct OutputEventRequestError

type OutputEventAPIError struct {
    Error error
} // <-- struct OutputEventAPIError

// Terminate the input channel reading loop. For internal use only
type input_event_terminate struct{}

type InputEventSendMessage struct {
    Message string
} // <-- struct InputEventSendMessage
