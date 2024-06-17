package sniff

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"syscall"
)

const Action = "sniff"

type FastCGITrame struct {
	Requests  string `json:"requests"`
	Responses string `json:"responses"`
}

func Run(args []string) error {
	phpFpmAddr := "127.0.0.1:9000"
	proxyAddr := "127.0.0.1:9001"
	help := false
	fs := flag.NewFlagSet(Action, flag.ContinueOnError)
	fs.StringVar(&phpFpmAddr, "forward-to", phpFpmAddr, "forward to fpm server at")
	fs.StringVar(&proxyAddr, "listen", proxyAddr, "proxy fastcgi listen to")
	fs.BoolVar(&help, "help", help, "print cmd help")
	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("cannot parse argument : %w", err)
	}
	if help {
		fs.PrintDefaults()
		return nil
	}

	listener, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		return fmt.Errorf("Error starting TCP proxy: %w", err)
	}
	log.Printf("Proxy listening on %s, forwarding to %s", proxyAddr, phpFpmAddr)
	runProxy(listener, phpFpmAddr)
	return nil
}

func runProxy(listener net.Listener, phpFpmAddr string) {
	var wg sync.WaitGroup
	for {
		log.Printf("waiting for tcp client")
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer clientConn.Close()
			log.Printf("handling tcp client")
			handleConnection(clientConn, phpFpmAddr)
		}()
	}
	listener.Close()
	wg.Wait()

}

func handleConnection(clientConn net.Conn, phpFpmAddr string) {
	serverConn, err := net.Dial("tcp", phpFpmAddr)
	if err != nil {
		log.Fatalf("Error connecting to PHP-FPM: %v", err)
	}
	defer serverConn.Close()
	log.Printf("connected to php-fpm")

	for {
		log.Printf("reading from tcp client")
		request, err := readFromConn(clientConn)
		if err != nil {
			if errors.Is(err, syscall.ECONNRESET) {
				log.Printf("Connection reset by peer: %v", err)
				break
			}
			if err != io.EOF {
				log.Fatalf("Error reading request from client: %v", err)
			}
			break
		}
		log.Println("Requests", base64.StdEncoding.EncodeToString(request))

		log.Printf("writing to php-fpm")
		_, err = serverConn.Write(request)
		if err != nil {
			log.Fatalf("Error sending request to server: %v", err)
			break
		}

		log.Printf("reading from php-fpm")
		response, err := readFromConn(serverConn)
		if err != nil {
			if err != io.EOF {
				log.Fatalf("Error reading response from server: %v", err)
			}
			break
		}

		log.Println("Responses", base64.StdEncoding.EncodeToString(response))

		log.Printf("writting back to tcp client")
		_, err = clientConn.Write(response)
		if err != nil {
			log.Fatalf("Error sending response to client: %v", err)
			break
		}

	}
}

func readFromConn(conn net.Conn) ([]byte, error) {
	buf := make([]byte, 4096)
	var data []byte

	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				data = append(data, buf[:n]...)
				return data, nil
			}
			return nil, err
		}
		data = append(data, buf[:n]...)
		if n < 4096 {
			break
		}
	}
	return data, nil
}
