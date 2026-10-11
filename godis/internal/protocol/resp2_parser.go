package protocol

import (
	"bytes"
	"strconv"
)

type Resp2Parser struct{}

func NewResp2Parser() *Resp2Parser {
	return &Resp2Parser{}
}

func (r *Resp2Parser) ParseCommand(data []byte) (args [][]byte, consumed int, err error) {
	if len(data) == 0 {
		return nil, 0, ErrIncomplete
	}

	var parsedCommand [][]byte

	var argumentCount int

	if data[0] != '*' {
		return nil, 0, ErrMalformed
	}

	argumentHeaderLineEnd := bytes.Index(data, []byte("\r\n"))
	if argumentHeaderLineEnd == -1 {
		return nil, 0, ErrIncomplete
	}

	argumentCount, err = strconv.Atoi(string(data[1:argumentHeaderLineEnd]))
	if err != nil || argumentCount <= 0 {
		return nil, 0, ErrMalformed
	}

	offset := argumentHeaderLineEnd + 2
	for i := 0; i < argumentCount; i++ {
		if offset >= len(data) {
			return nil, 0, ErrIncomplete
		}

		if data[offset] != '$' {
			return nil, 0, ErrMalformed
		}

		relativeEnd := bytes.Index(data[offset:], []byte("\r\n"))
		if relativeEnd == -1 {
			return nil, 0, ErrIncomplete
		}

		lineEnd := offset + relativeEnd
		lengthBytes := data[offset+1 : lineEnd]

		bulkStringLength, err := strconv.Atoi(string(lengthBytes))
		if err != nil || bulkStringLength < 0 {
			return nil, 0, ErrMalformed
		}

		if bulkStringLength > maxBulkStringLength {
			return nil, 0, ErrMalformed
		}

		startBulkString := lineEnd + 2
		if bulkStringLength > len(data)-startBulkString {
			return nil, 0, ErrIncomplete
		}

		endBulkString := startBulkString + bulkStringLength
		bulkString := data[startBulkString:endBulkString]

		if len(data)-endBulkString < 2 {
			return nil, 0, ErrIncomplete
		}

		if data[endBulkString] != '\r' || data[endBulkString+1] != '\n' {
			return nil, 0, ErrMalformed
		}

		parsedCommand = append(parsedCommand, bulkString)

		offset = endBulkString + 2
	}

	return parsedCommand, offset, nil
}
