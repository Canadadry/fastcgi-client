package server

import (
	"io"
	"log"
	"net"
	"sync"
)

type Hanlder func(clientConn io.ReadWriter) error

func Run(done <-chan struct{}, listener net.Listener, handler Hanlder) {
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
			log.Println("Context done, stopping listener accept loop")
			wg.Wait()
			return
		case clientConn := <-connChan:
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer clientConn.Close()
				log.Printf("Handling TCP client")
				err := handler(clientConn)
				if err != nil {
					log.Println(err)
				}
			}()
		case err := <-errChan:
			log.Printf("Error accepting connection: %v", err)
		}
	}
}
