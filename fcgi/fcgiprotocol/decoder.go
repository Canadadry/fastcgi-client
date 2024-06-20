package fcgiprotocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Request struct {
	ReqId uint16
	Env   map[string]string
	Stdin []byte
}

func DecodeRequest(input []Record) (Request, error) {
	decoded := Request{}
	var err error
	if input[0].Header.Type != FCGI_BEGIN_REQUEST {
		return Request{}, fmt.Errorf(
			"request should start with packet 'begin' got %v",
			input[0].Header.Type,
		)
	}
	decoded.ReqId = input[0].Header.Id
	envContent := []byte{}
	for _, r := range input[1:] {
		switch r.Header.Type {
		case FCGI_PARAMS:
			envContent = append(envContent, r.Content()...)
		case FCGI_STDIN:
			decoded.Stdin = append(decoded.Stdin, r.Content()...)
		}
	}
	decoded.Env, err = decodeEnv(bytes.NewReader(envContent))
	if err != nil {
		return decoded, fmt.Errorf("cannot decode param %w", err)
	}
	return decoded, nil
}

func decodeEnv(r io.Reader) (map[string]string, error) {
	pairs := make(map[string]string)
	b := make([]byte, 4)

	for {
		// Read the key length
		n, err := r.Read(b[:1])
		if err != nil {
			if err == io.EOF && n == 0 {
				break
			}
			return nil, err
		}

		var keyLen uint32
		if b[0] > 127 {
			// Key length is encoded in 4 bytes
			if _, err := io.ReadFull(r, b[1:4]); err != nil {
				return nil, err
			}
			binary.BigEndian.PutUint32(b[:4], binary.BigEndian.Uint32(b[:4])&^(1<<31))
			keyLen = binary.BigEndian.Uint32(b[:4])
		} else {
			// Key length is encoded in 1 byte
			keyLen = uint32(b[0])
		}

		// Read the value length
		if _, err := io.ReadFull(r, b[:1]); err != nil {
			return nil, err
		}

		var valueLen uint32
		if b[0] > 127 {
			// Value length is encoded in 4 bytes
			if _, err := io.ReadFull(r, b[1:4]); err != nil {
				return nil, err
			}
			binary.BigEndian.PutUint32(b[:4], binary.BigEndian.Uint32(b[:4])&^(1<<31))
			valueLen = binary.BigEndian.Uint32(b[:4])
		} else {
			// Value length is encoded in 1 byte
			valueLen = uint32(b[0])
		}

		// Read the key
		key := make([]byte, keyLen)
		if _, err := io.ReadFull(r, key); err != nil {
			return nil, err
		}

		// Read the value
		value := make([]byte, valueLen)
		if _, err := io.ReadFull(r, value); err != nil {
			return nil, err
		}

		pairs[string(key)] = string(value)
	}

	return pairs, nil
}

type Response struct {
	StatusCode int
	Headers    map[string]string
	Stdout     string
}

func ParseResponse(content string) (Response, error) {

	parts := strings.SplitN(content, "\r\n\r\n", 2)

	if len(parts) < 2 {
		return Response{StatusCode: 502}, fmt.Errorf("cannot parse response, content must have 2 part got %v", len(parts))
	}

	rsp := Response{
		StatusCode: 200,
		Headers:    ParseHeader(parts[0]),
		Stdout:     parts[1],
	}

	if st, ok := rsp.Headers["Status"]; ok {
		rsp.StatusCode, _ = strconv.Atoi(st[0:3])
	}

	return rsp, nil
}

func ParseHeader(content string) map[string]string {
	headers := map[string]string{}
	headerParts := strings.Split(content, "\r\n")
	for _, line := range headerParts {
		lineParts := strings.SplitN(line, ":", 2)
		if len(lineParts) < 2 {
			continue
		}
		headers[lineParts[0]] = strings.TrimSpace(lineParts[1])
	}
	return headers
}
