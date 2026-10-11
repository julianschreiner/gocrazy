package protocol

type ParserError string

func (e ParserError) Error() string {
	return string(e)
}

const (
	ErrIncomplete ParserError = "incomplete RESP message"
	ErrMalformed  ParserError = "malformed RESP message"
)

const maxBulkStringLength = 512 * 1024 * 1024

type Parser interface {
	ParseCommand(data []byte) (args [][]byte, consumed int, err error)
}
