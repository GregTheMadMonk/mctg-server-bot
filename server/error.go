package server

const (
    ERR_RUNNING = iota
    ERR_ETYPE   = iota
    ERR_TRAWJS  = iota
    ERR_USER    = iota
)

type Error struct {
    Type    uint
}

func (self *Error) Error() string {
    switch (self.Type) {
    case ERR_RUNNING:
        return "Server is already running"
    case ERR_ETYPE:
        return "Unknown event type"
    case ERR_TRAWJS:
        return "Cannot serialize /tellraw message argument"
    case ERR_USER:
        return "User error"
    default:
        return "Unknown server error"
    }
} // <-- func Error::Error()
