package server

import (
	"fmt"
	"io"
)

type DialFunc func() (io.ReadWriteCloser, error)

func Proxy[T any](dial DialFunc, clientToServer, serverToClient Pipe[T], printf func(msg string, args ...interface{})) Hanlder {
	return func(clientConn io.ReadWriter) error {
		serverConn, err := dial()
		if err != nil {
			return fmt.Errorf("error connecting to PHP-FPM: %w", err)
		}
		defer serverConn.Close()
		printf("connected to server\n")

		err = clientToServer.Run(clientConn, serverConn, "request", printf)
		if err != nil {
			return err
		}

		printf("finish writing to server, waiting for response\n")
		err = serverToClient.Run(serverConn, clientConn, "response", printf)
		if err != nil {
			return err
		}
		return nil
	}
}
