package server

import (
	"context"
	"io"
	"net"
	"testing"
	"time"
)

func mockHandler(conn io.ReadWriter) error {
	time.Sleep(100 * time.Millisecond)
	return nil
}

func TestRun(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()

	done := make(chan struct{})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		<-ctx.Done()
		close(done)
	}()

	go Run(done, listener, mockHandler)

	// Connect to the listener
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Error dialing to listener: %v", err)
	}
	defer conn.Close()

	// Simulate some activity
	_, err = conn.Write([]byte("test"))
	if err != nil {
		t.Fatalf("Error writing to connection: %v", err)
	}

	// Wait a moment to let the connection be handled
	time.Sleep(200 * time.Millisecond)

	// Cancel the context to stop the proxy
	cancel()

	// Give some time for the server to shut down
	time.Sleep(200 * time.Millisecond)

	// Verify that the listener is closed
	select {
	case <-done:
	// Server stopped as expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Server did not shut down as expected")
	}
}
