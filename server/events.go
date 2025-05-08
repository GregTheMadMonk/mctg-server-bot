package server

type OutputEventLog struct {
    Message string
} // <-- struct OutputEventLog

type OutputEventMessage struct {
    Tellraw  bool // Redirect this message to the server too
    Username string
    Message  string
} // <-- struct OutputEventMessage

type OutputEventPlayerDeath struct {
    Username string
    Message  string
} // <-- struct OutputEventPlayerDeath

type OutputEventPlayerAchievement struct {
    Username    string
    Achievement string
} // <-- struct OutputEventPlayerAchievement

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

type OutputEventListTeams struct {
    Teams []string
} // <-- struct OutputEventListTeams

type OutputEventTeamMapping struct {
    Mapping TeamMapping
} // <-- struct OutputEventTeamMapping

type OutputEventError struct {
    Error error
} // <-- struct OutputEventError

type OutputEventExit struct {
    // -2 if can't get ExitCode()
    ExitCode int
} // <-- struct OutputEventExit

// Terminate the input channel reading loop. For internal use onlu
type input_event_terminate struct {}

type input_event_fetch_teams struct {}

type input_event_req_team struct {
    Team string
} // <-- struct input_event_req_team

type input_event_update_team struct {
    Team      string
    Usernames []string
} // <-- struct input_event_update_team

type InputEventChat struct {
    Telegram bool
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

type InputEventBindRename struct {
    Username    string
    DisplayName string
} // <-- struct InputEventBindTelegramUser

type InputEventListPlayers struct {}

type InputEventKillServer struct {}
