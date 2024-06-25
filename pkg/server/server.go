package server

import (
	"io"
	"net"
	"sync"
)

type Hanlder func(clientConn io.ReadWriter) error

func Run(done <-chan struct{}, listener net.Listener, handler Hanlder, printf func(msg string, args ...interface{})) {
	var wg sync.WaitGroup

	connChan := make(chan net.Conn)
	errChan := make(chan error)

	go func() {
		for {
			clientConn, err := listener.Accept()
			if err != nil {
				errChan <- err
				return
			}
			connChan <- clientConn
		}
	}()

	for {
		select {
		case <-done:
			printf("context done, stopping listener accept loop")
			wg.Wait()
			return
		case clientConn := <-connChan:
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer clientConn.Close()
				printf("handling new TCP client\n")
				err := handler(clientConn)
				if err != nil {
					println("error handeling client %v\n", err)
				}
			}()
		case err := <-errChan:
			printf("rrror accepting connection: %v\n", err)
		}
	}
}
