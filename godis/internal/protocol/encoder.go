package protocol

type Encoder interface {
	// +OK\r\n
	EncodeSimpleString(value string) []byte

	// -ERR unknown command\r\n
	EncodeError(message string) []byte

	// :42\r\n
	EncodeInteger(value int64) []byte

	// $5\r\nhello\r\n
	EncodeBulkString(value []byte) []byte

	// $-1\r\n
	EncodeNullBulkString() []byte

	// Encode an array of bulk strings.
	EncodeBulkStringArray(values [][]byte) []byte
}
