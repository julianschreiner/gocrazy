package protocol

import (
	"strconv"
)

const CRLF = "\r\n"

type Resp2Encoder struct{}

func NewResp2Encoder() *Resp2Encoder {
	return &Resp2Encoder{}
}

func (r *Resp2Encoder) EncodeSimpleString(value string) []byte {
	return []byte("+" + value + CRLF)
}

func (r *Resp2Encoder) EncodeError(message string) []byte {
	return []byte("-" + message + CRLF)
}

func (r *Resp2Encoder) EncodeInteger(value int64) []byte {
	return []byte(":" + strconv.FormatInt(value, 10) + CRLF)
}

func (r *Resp2Encoder) EncodeBulkString(value []byte) []byte {
	header := "$" + strconv.Itoa(len(value)) + CRLF

	result := []byte(header)
	result = append(result, value...)
	result = append(result, CRLF...)

	return result
}

func (r *Resp2Encoder) EncodeNullBulkString() []byte {
	return []byte("$-1" + CRLF)
}

func (r *Resp2Encoder) EncodeBulkStringArray(values [][]byte) []byte {
	lenValues := len(values)
	header := "*" + strconv.Itoa(len(values)) + CRLF

	result := []byte(header)

	for i := range lenValues {
		bulkStringByteCount := strconv.Itoa(len(values[i]))
		bulkStringContent := values[i]

		result = append(result, []byte("$"+bulkStringByteCount+CRLF)...)
		result = append(result, []byte(bulkStringContent)...)
		result = append(result, []byte(CRLF)...)
	}

	return result
}
