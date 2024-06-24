package server

import (
	"fmt"
	"io"
	"log"
)

type DialFunc func() (io.ReadWriteCloser, error)

func Proxy[T any](dial DialFunc, clientToServer, serverToClient Pipe[T]) Hanlder {
	return func(clientConn io.ReadWriter) error {
		serverConn, err := dial()
		if err != nil {
			return fmt.Errorf("error connecting to PHP-FPM: %w", err)
		}
		defer serverConn.Close()
		log.Printf("connected to php-fpm")

		err = clientToServer.Run(clientConn, serverConn, "request")
		if err != nil {
			return err
		}

		err = serverToClient.Run(serverConn, clientConn, "response")
		if err != nil {
			return err
		}
		return nil
	}
}
