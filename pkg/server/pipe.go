package server

import (
	"encoding/json"
	"fmt"
	"io"
)

type Pipe[T any] struct {
	Reader  func(r io.Reader) (T, error)
	Writer  func(w io.Writer, data T) error
	Decoder func(data T) (interface{}, error)
}

func (p *Pipe[T]) Run(r io.Reader, w io.Writer, prefix string, printf func(msg string, args ...interface{})) error {
	data, err := p.Reader(r)
	if err != nil {
		return fmt.Errorf("cannot read %s : %w", prefix, err)
	}
	jsonRawRecs, err1 := json.Marshal(data)
	if err1 != nil {
		return fmt.Errorf("cannot marshal raw %s : %w", prefix, err1)
	}
	printf("%s read raw %s\n", prefix, string(jsonRawRecs))
	if p.Decoder != nil {
		decoded, err := p.Decoder(data)
		if err != nil {
			return fmt.Errorf("cannot decode %s : %w", prefix, err)
		}
		jsonReqs, err := json.Marshal(decoded)
		if err != nil {
			return fmt.Errorf("cannot marshal decoded %s : %w", prefix, err)
		}
		printf("decoded %s %s\n", prefix, string(jsonReqs))
	}

	printf("writing back %s\n", prefix)
	return p.Writer(w, data)
}
