package command

type GodisCommand string

const (
	CommandPing   GodisCommand = "PING"
	CommandEcho   GodisCommand = "ECHO"
	CommandSet    GodisCommand = "SET"
	CommandGet    GodisCommand = "GET"
	CommandDel    GodisCommand = "DEL"
	CommandExists GodisCommand = "EXISTS"
	CommandIncr   GodisCommand = "INCR"
	CommandKeys   GodisCommand = "KEYS"
)

type GodisCommandError string

const (
	ErrUnknownCommand             = "ERR unknown command"
	ErrEmptyCommand               = "ERR empty command"
	ErrWrongNumberOfArgs          = "ERR wrong number of arguments for '%s' command"
	ErrValueNoIntegerOrOutOfRange = "ERR value is not an integer or out of range"
)
