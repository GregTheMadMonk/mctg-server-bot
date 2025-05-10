// serverctl.go
// Run and control the minecraft server
package server

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os/exec"
    "regexp"
    "strings"
)

const RENAME_TEAM_PFX = "__internal_rename_"

// Server config
type Config struct {
    // Command-line to run the Minecraft server
    Cmdline []string `json:"cmdline"`
    // Log lines to store in RAM
    LogLines uint `json:"log_lines"`
} // <-- struct Config

// tellraw command
type tellraw_cmd struct {
    Text  string `json:"text"`
    Color string `json:"color"`
}

// Handle for a single Minecraft server instance
type Handle struct {
    // Server handler config
    config Config
    // Process command handle
    cmd *exec.Cmd
    // A list of active players
    players_online []string
    // Channel with the server's output
    out chan any
    // Channel with the server's input
    in chan any

    // Teams on the server
    teams TeamMapping

    // Child process stdout
    stdout *io.ReadCloser
    // Child process stdin
    stdin *io.WriteCloser

    // If true, the server will try to restart when cmd exits
    TryRestart bool
} // <-- struct Handle

// The player has joined the server
func (self *Handle) push_player(username string) {
    found := false
    for _, player := range self.players_online {
        found = found || (player == username)
    }
    if !found {
        self.players_online = append(self.players_online, username)
    }
} // <-- Handle::push_player(username)

// The player has left the server
func (self *Handle) pop_player(username string) {
    for i, player := range self.players_online {
        if player == username {
            self.players_online = append(
                self.players_online[:i],
                self.players_online[i+1:]...,
            )
            return
        }
    }
} // <-- func Handle::pop_player(username)

// Get the server's output channel
func (self *Handle) Out() <-chan any {
    return self.out
} // <-- Handle::Out()

// Get the server's input channel
func (self *Handle) In() chan<- any {
    return self.in
} // <-- Handle::Out()

// Handle writing to the server's stdin
func (self *Handle) handle_stdin() {
    say := func(cmd []tellraw_cmd) {
        if str, err := json.Marshal(cmd); err == nil {
            fmt.Fprintf(*self.stdin, "/tellraw @a %s\n", str)
        } else {
            self.out <- OutputEventError{&Error{ERR_TRAWJS}}
        }
    } // <-- say(cmd)

    make_tellraw := func(usr string, t string, tg bool, e bool) []tellraw_cmd {
        ret := []tellraw_cmd{{Text: "@", Color: "gold"}}

        if tg {
            ret[0].Color = "blue"
        }

        ret = append(ret, tellraw_cmd{Text: usr, Color: "yellow"})
        if e {
            ret = append(
                ret,
                tellraw_cmd{Text: " corrects", Color: "dark_gray"},
            )
        }
        ret = append(ret, tellraw_cmd{Text: ": ", Color: "white"})
        ret = append(ret, tellraw_cmd{Text: t, Color: "white"})

        return ret
    } // <-- make_tellraw(user, text, tg)

    username := func(usr string) string {
        team := fmt.Sprintf("%s%s", RENAME_TEAM_PFX, usr)
        for _, display := range self.teams.TeamPlayers(team) {
            return display
        }
        return usr
    } // <-- username(usr)

    make_tellraw_colored := func(ct []ColoredSymbol) []tellraw_cmd {

        ret := []tellraw_cmd{}
        for _, symbol := range ct {
            ret = append(ret, tellraw_cmd{Text: string(symbol.Symbol), Color: symbol.Color})
        }

        return ret
    } // <-- make_tellraw_colored(user, text, tg)

    for {
        ie, open := <-self.in
        if !open {
            continue
        }

        // cmd == nil means that all IO to the child process is invalid
        if self.cmd == nil {
            if _, is_term := ie.(input_event_terminate); is_term {
                break
            } else {
                continue // Wait for terminate event
            }
        }

        switch event := ie.(type) {
        case input_event_fetch_teams:
            fmt.Fprintf(*self.stdin, "/team list\n")
        case input_event_req_team:
            fmt.Fprintf(*self.stdin, "/team list %s\n", event.Team)
        case input_event_update_team:
            self.teams.Data = append(
                self.teams.Data, Team{
                    Name:      event.Team,
                    Usernames: event.Usernames,
                },
            )
            log.Println(self.teams)
        case InputEventChat:
            for _, l := range strings.Split(event.Message, "\n") {
                say(
                    make_tellraw(
                        username(event.Username), l, event.Telegram, false,
                    ),
                )
            }
        case InputEventEditChat:
            for _, l := range strings.Split(event.Message, "\n") {
                say(make_tellraw(username(event.Username), l, true, true))
            }
        case InputEventBindRename:
            team_name := fmt.Sprintf("%s%s", RENAME_TEAM_PFX, event.Username)
            old_teams := self.teams.PlayerTeams(event.DisplayName)
            for _, team := range old_teams {
                if !strings.HasPrefix(team, RENAME_TEAM_PFX) {
                    continue
                }

                if team == team_name {
                    continue
                }

                self.out <- OutputEventError{&Error{ERR_USER}}
                break
            }

            fmt.Fprintf(*self.stdin, "/team remove %s\n", team_name)
            fmt.Fprintf(*self.stdin, "/team add %s\n", team_name)
            fmt.Fprintf(
                *self.stdin,
                "/team join %s %s\n",
                team_name,
                event.DisplayName,
            )
            fmt.Fprintf(*self.stdin, "/team list\n")
        case InputEventListPlayers:
            self.out <- OutputEventListPlayers{
                PlayersOnline: self.players_online,
            }
        case InputEventCommand:
            for _, l := range strings.Split(event.Command, "\n") {
                fmt.Fprintln(*self.stdin, l)
            }
        case InputEventKillServer:
            self.TryRestart = false
            fmt.Fprintf(*self.stdin, "/stop\n")
        case InputEventColoredChat:
            say(make_tellraw(event.Username, "", event.Telegram, false))
            for _, l := range event.ColoredMessage {
                say(make_tellraw_colored(l))
            }
        default:
            self.out <- OutputEventError{&Error{ERR_ETYPE}}
        }
    }
    self.stdin = nil
    log.Println("Exit server.Handle::handle_stdin()")
} // <-- Handle::handle_stdin(stdin)

// Handle reading from the server's stdout
func (self *Handle) handle_stdout() {
    reader := bufio.NewReader(*self.stdout)

    // I'm sorry for what's about to follow, my precious 80-column line limit :(
    message_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/MinecraftServer\]:( \[Not Secure\])* \<([A-Za-z0-9_\.]+)\> (.*)\r*\n$`,
    )
    raw_message_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[co.gr.mc.MCTGMod\/\]: CHAT([A-Za-z0-9_\.]+)(.*)\r*\n$`,
    )
    death_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[co.gr.mc.MCTGMod\/\]: DEATH([A-Za-z0-9_\.]+)(.*)\r*\n$`,
    )
    teams_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/MinecraftServer\]: There are ([0-9]+) team\(s\): (.+)\r*\n$`,
    )
    team_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/MinecraftServer\]: Team (.+) has ([0-9]+) member\(s\): (.+)\r*\n$`,
    )
    joined_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/MinecraftServer\]: ([A-Za-z0-9_\.]+) joined the game\r*\n$`,
    )
    left_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/MinecraftServer\]: ([A-Za-z0-9_\.]+) left the game\r*\n$`,
    )
    done_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/DedicatedServer\]: Done \([0-9]\.[0-9]+s\)! For help, type "help"\r*\n$`,
    )
    achievement_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/MinecraftServer\]: ([A-Za-z0-9_\.]+) has made the advancement \[(.*)\]\r*\n$`,
    )

    for {
        if self.cmd == nil {
            break
        }

        if str, err := reader.ReadString('\n'); err == nil {
            self.out <- OutputEventLog{str}

            if sm := message_r.FindStringSubmatch(str); sm != nil {
                self.out <- OutputEventMessage{
                    Tellraw:  false,
                    Username: sm[2],
                    Message:  sm[3],
                }
            } else if sm := raw_message_r.FindStringSubmatch(str); sm != nil {
                self.out <- OutputEventLog{
                    fmt.Sprintf("%s: %s\n", sm[1], sm[2]),
                }
                self.out <- OutputEventMessage{
                    Tellraw:  true,
                    Username: sm[1],
                    Message:  sm[2],
                }
            } else if sm := death_r.FindStringSubmatch(str); sm != nil {
                self.out <- OutputEventPlayerDeath{
                    Username: sm[1],
                    Message:  sm[2],
                }
            } else if sm := teams_r.FindStringSubmatch(str); sm != nil {
                self.teams = TeamMapping{}
                for _, team := range strings.Split(sm[2], ", ") {
                    self.in <- input_event_req_team{Team: team[1 : len(team)-1]}
                }
            } else if sm := team_r.FindStringSubmatch(str); sm != nil {
                self.in <- input_event_update_team{
                    Team:      sm[1][1 : len(sm[1])-1],
                    Usernames: strings.Split(sm[3], ", "),
                }
            } else if sm := joined_r.FindStringSubmatch(str); sm != nil {
                self.push_player(sm[1])
                self.out <- OutputEventPlayerJoined{
                    Username: sm[1],
                }
            } else if sm := left_r.FindStringSubmatch(str); sm != nil {
                self.pop_player(sm[1])
                self.out <- OutputEventPlayerLeft{
                    Username: sm[1],
                }
            } else if sm := achievement_r.FindStringSubmatch(str); sm != nil {
                self.out <- OutputEventPlayerAchievement{
                    Username:    sm[1],
                    Achievement: sm[2],
                }
            } else if sm := done_r.FindStringSubmatch(str); sm != nil {
                self.out <- OutputEventServerLoaded{}
            }
        } else {
            self.out <- OutputEventError{err}
        }
    }

    self.stdout = nil
    log.Println("Exit server.Handle::handle_stdout()")
} // <-- Handle::handle_stdout(stdout)

// Monitor the server's state
func (self *Handle) watch_child() {
    self.out <- OutputEventExit{
        func() int {
            // Wait for the child process to finish
            err := self.cmd.Wait()
            self.cmd = nil

            // Send a dummy input event to ensure loop reaches termination
            self.in <- input_event_terminate{}

            // Wait for the IO handlers to finish
            for {
                if self.stdin == nil && self.stdout == nil {
                    break
                }
            }

            if err != nil {
                if exiterr, ok := err.(*exec.ExitError); ok {
                    return exiterr.ExitCode()
                }
                return -2
            }
            return 0
        }(),
    }
} // <-- Handle::watch_child()

// Check if the process is stopped
func (self *Handle) IsRunning() bool { return self.cmd != nil }

// Run the server command-line and attach reader and writer routines
// This call is non-blocking and returns nil on success, error on failure
func (self *Handle) Start() error {
    if self.cmd != nil {
        return &Error{ERR_RUNNING}
    }

    self.cmd = exec.Command(self.config.Cmdline[0], self.config.Cmdline[1:]...)
    pipe_in, in_err := self.cmd.StdinPipe()
    if in_err != nil {
        return in_err
    }
    pipe_out, out_err := self.cmd.StdoutPipe()
    if out_err != nil {
        return out_err
    }

    // Start the process
    if err := self.cmd.Start(); err != nil {
        return err
    }

    self.TryRestart = true

    // Handle the process IO
    self.stdin = &pipe_in
    self.stdout = &pipe_out

    // Update the teams
    fmt.Fprintf(*self.stdin, "/team list\n")

    go self.handle_stdin()
    go self.handle_stdout()

    // Keepalive cycle
    go self.watch_child()

    return nil
} // <-- Handle::Start()

func (self *Handle) ReverseRename(username string) string {
    for _, team := range self.teams.PlayerTeams(username) {
        if strings.HasPrefix(team, RENAME_TEAM_PFX) {
            return team[len(RENAME_TEAM_PFX):]
        }
    }
    return username
} // <-- Handle::ReverseRename(username)

// Create server handle from the config
func MakeHandle(server_cfg Config) Handle {
    return Handle{
        config:     server_cfg,
        cmd:        nil,
        out:        make(chan any),
        in:         make(chan any),
        TryRestart: true,
    }
} // <-- MakeHandle(server_cfg)
