package server

import (
	"bytes"
	"errors"
	"io"
	"log"
	"strings"
	"testing"
)

type mockConn struct {
	io.ReadWriteCloser
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
}

func newMockConn(readData string) *mockConn {
	return &mockConn{
		readBuf:  bytes.NewBufferString(readData),
		writeBuf: &bytes.Buffer{},
	}
}

func (m *mockConn) Read(p []byte) (n int, err error) {
	return m.readBuf.Read(p)
}

func (m *mockConn) Write(p []byte) (n int, err error) {
	return m.writeBuf.Write(p)
}

func (m *mockConn) Close() error {
	return nil
}

func mockDialFunc() (io.ReadWriteCloser, error) {
	return newMockConn("response data"), nil
}

func mockFailDialFunc() (io.ReadWriteCloser, error) {
	return nil, errors.New("dial error")
}

func WriteAll(w io.Writer, data []byte) error {
	_, err := w.Write(data)
	return err
}

func WriteAllError(w io.Writer, data []byte) error {
	return errors.New("pipe error")
}

func TestHandleConnection_Success(t *testing.T) {
	out := &bytes.Buffer{}
	clientConn := newMockConn("request data")
	l := log.New(out, "", 0)
	handler := Proxy[[]byte](mockDialFunc,
		Pipe[[]byte]{Reader: io.ReadAll, Writer: WriteAll},
		Pipe[[]byte]{Reader: io.ReadAll, Writer: WriteAll},
		func(msg string, args ...interface{}) { l.Printf(msg, args...) },
	)

	err := handler(clientConn)
	if err != nil {
		t.Fatalf("HandleConnection returned an error: %v", err)
	}

	expected := "response data"
	if clientConn.writeBuf.String() != expected {
		t.Errorf("Expected client write buffer to be %q, but got %q", expected, clientConn.writeBuf.String())
	}
	expectedLog := strings.Join([]string{
		"connected to server",
		"request read raw \"cmVxdWVzdCBkYXRh\"",
		"writing back request",
		"finish writing to server, waiting for response",
		"response read raw \"cmVzcG9uc2UgZGF0YQ==\"",
		"writing back response",
		"",
	}, "\n")
	if out.String() != expectedLog {
		t.Errorf("Expected log \n%s\n, but got \n%s\n", expectedLog, out.String())
	}
}

func TestHandleConnection_DialError(t *testing.T) {
	out := &bytes.Buffer{}
	clientConn := newMockConn("request data")
	l := log.New(out, "", 0)
	handler := Proxy(mockFailDialFunc,
		Pipe[[]byte]{Reader: io.ReadAll, Writer: WriteAll},
		Pipe[[]byte]{Reader: io.ReadAll, Writer: WriteAll},
		func(msg string, args ...interface{}) { l.Printf(msg, args...) },
	)

	err := handler(clientConn)
	if err == nil || err.Error() != "error connecting to PHP-FPM: dial error" {
		t.Fatalf("Expected dial error, but got: %v", err)
	}
}

func TestHandleConnection_PipeRunError(t *testing.T) {
	out := &bytes.Buffer{}
	clientConn := newMockConn("request data")
	l := log.New(out, "", 0)
	handler := Proxy(mockDialFunc,
		Pipe[[]byte]{Reader: io.ReadAll, Writer: WriteAll},
		Pipe[[]byte]{Reader: io.ReadAll, Writer: WriteAllError},
		func(msg string, args ...interface{}) { l.Printf(msg, args...) },
	)

	err := handler(clientConn)
	if err == nil || err.Error() != "pipe error" {
		t.Fatalf("Expected pipe error, but got: %v", err)
	}
}
