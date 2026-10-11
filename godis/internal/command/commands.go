package command

type GodisCommand string

const (
	CommandPing GodisCommand = "PING"
	CommandEcho GodisCommand = "ECHO"
)

type GodisCommandError string

const (
	ErrUnkownCommand = "ERR unknown command"
	ErrEmptyCommand  = "ERR empty command"
)
