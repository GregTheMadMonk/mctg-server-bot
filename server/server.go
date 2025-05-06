// serverctl.go
// Run and control the minecraft server
package server

import (
    "bufio"
    "fmt"
    "io"
    "log"
    "os/exec"
    "regexp"
    "strings"
)

// Server config
type Config struct {
    // Command-line to run the Minecraft server
    Cmdline  []string `json:"cmdline"`
    // Log lines to store in RAM
    LogLines uint     `json:"log_lines"`
} // <-- struct Config

// Handle for a single Minecraft server instance
type Handle struct {
    // Server handler config
    config         Config
    // Process command handle
    cmd            *exec.Cmd
    // A list of active players
    players_online []string
    // Channel with the server's output
    out            chan any
    // Channel with the server's input
    in             chan any

    // Child process stdout
    stdout         *io.ReadCloser
    // Child process stdin
    stdin          *io.WriteCloser

    // If true, the server will try to restart when cmd exits
    TryRestart     bool
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
                self.players_online[i+1:]...
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
        case InputEventChat:
            for _, l := range strings.Split(event.Message, "\n") {
                fmt.Fprintf(
                    *self.stdin,
                    "/say §6@%s§f: %s\n",
                    event.Username,
                    l,
                )
            }
        case InputEventEditChat:
            for _, l := range strings.Split(event.Message, "\n") {
                fmt.Fprintf(
                    *self.stdin,
                    "/say §6@%s§8 corrects§f: %s\n",
                    event.Username,
                    l,
                )
            }
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
        default:
            self.out <- OutputEventError{ &Error{ ERR_ETYPE } }
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
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/MinecraftServer\]:( \[Not Secure\])* \<([A-Za-z0-9_\.]+)\> (.*)\n$`,
    )
    joined_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/MinecraftServer\]: ([A-Za-z0-9_\.]+) joined the game\n$`,
    )
    left_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/MinecraftServer\]: ([A-Za-z0-9_\.]+) left the game\n$`,
    )
    done_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/DedicatedServer\]: Done \([0-9]\.[0-9]+s\)! For help, type "help"\n$`,
    )
    achievement_r := regexp.MustCompile(
        `^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\] \[Server thread\/INFO\] \[minecraft\/MinecraftServer\]: ([A-Za-z0-9_\.]+) has made the advancement \[(.*)\]\n$`,
    )

    for {
        if self.cmd == nil {
            break
        }

        if str, err := reader.ReadString('\n'); err == nil {
            self.out <- OutputEventLog{ str }

            if sm := message_r.FindStringSubmatch(str); sm != nil {
                self.out <- OutputEventMessage{
                    Username: sm[2],
                    Message:  sm[3],
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
            self.out <- OutputEventError{ err }
        }
    }

    self.stdout = nil
    log.Println("Exit server.Handle::handle_stdout()")
} // <-- Handle::handle_stdout(stdout)

// Monitor the server's state
func (self *Handle) watch_child() {
    self.out <- OutputEventExit{
        func () int {
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
        } (),
    }
} // <-- Handle::watch_child()

// Check if the process is stopped
func (self *Handle) IsRunning() bool { return self.cmd != nil }

// Run the server command-line and attach reader and writer routines
// This call is non-blocking and returns nil on success, error on failure
func (self *Handle) Start() error {
    if self.cmd != nil {
        return &Error{ ERR_RUNNING }
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
    self.stdin  = &pipe_in
    self.stdout = &pipe_out
    go self.handle_stdin()
    go self.handle_stdout()

    // Keepalive cycle
    go self.watch_child()

    return nil
} // <-- Handle::Start()

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
