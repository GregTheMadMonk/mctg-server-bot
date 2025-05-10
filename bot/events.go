package bot

type OutputEventMessage struct {
    Username string
    Message  string
} // <-- struct OutputEventMessage

type OutputEventEditMessage struct {
    Username string
    Message  string
} // <-- sturct OutputEventEditMessage

type OutputEventImage struct {
    Username  string
    FilePath  string
    Extension string
    Content   []byte
} // <-- struct OutputEventImage

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
