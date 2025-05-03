package server

type OutputEventLog struct {
    Message string
} // <-- struct OutputEventLog

type OutputEventMessage struct {
    Username string
    Message  string
} // <-- struct OutputEventMessage

type OutputEventPlayerJoined struct {
    Username string
} // <-- struct OutputEventPlayerJoined

type OutputEventPlayerLeft struct {
    Username string
} // <-- struct OutputEventPlayerLeft

type OutputEventServerLoaded struct {}

type OutputEventListPlayers struct {
    PlayersOnline []string
} // <-- struct OutputEventListPlayers

type OutputEventError struct {
    Error error
} // <-- struct OutputEventError

type OutputEventExit struct {
    // -2 if can't get ExitCode()
    ExitCode int
} // <-- struct OutputEventExit

// Terminate the input channel reading loop. For internal use onlu
type input_event_terminate struct {}

type InputEventChat struct {
    Username string
    Message  string
} // <-- struct InputEventChat

type InputEventEditChat struct {
    Username string
    Message  string
} // <-- struct InputEventChat

type InputEventCommand struct {
    Command string
} // <-- struct InputEventCommand

type InputEventListPlayers struct {}

type InputEventKillServer struct {}
