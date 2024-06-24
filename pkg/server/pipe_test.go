package server

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"
)

func TestPipeRun(t *testing.T) {
	mockData := []byte{0, 1, 2, 3, 4, 5}

	mockReader := func(r io.Reader) ([]byte, error) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r)
		return buf.Bytes(), nil
	}

	mockWriter := func(w io.Writer, data []byte) error {
		_, err := w.Write(data)
		return err
	}

	mockDecoder := func(data []byte) (interface{}, error) {
		return string(data), nil
	}

	out := &bytes.Buffer{}
	pipe := Pipe[[]byte]{
		Reader:  mockReader,
		Writer:  mockWriter,
		Decoder: mockDecoder,
		Logger:  log.New(out, "", 0),
	}

	clientConn := bytes.NewBuffer(mockData)
	serverConn := &bytes.Buffer{}

	err := pipe.Run(clientConn, serverConn, "test")
	if err != nil {
		t.Fatalf("Run returned an error: %v", err)
	}

	if serverConn.String() != string(mockData) {
		t.Fatalf("Expected %s but got %s", string(mockData), serverConn.String())
	}

	expected := strings.Join([]string{
		"test read raw \"AAECAwQF\"",
		"decoded test \"\\u0000\\u0001\\u0002\\u0003\\u0004\\u0005\"",
		"writing back test",
		"",
	}, "\n")

	if out.String() != expected {
		t.Fatalf("Expected \n%s\n but got \n%s\n", expected, out.String())
	}
}
