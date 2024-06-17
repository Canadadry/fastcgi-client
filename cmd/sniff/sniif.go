package sniff

import (
	"app/fcgiclient"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
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
		reqs, err := readFullRequest(clientConn)
		if err != nil {
			log.Fatalf("cannot read request: %v", err)
			break
		}
		jsonReqs, err := json.Marshal(reqs)
		if err != nil {
			log.Fatalf("cannot encode request to json: %v", err)
			break
		}
		log.Println("Requests", string(jsonReqs))

		log.Printf("writing to php-fpm")
		for _, r := range reqs {
			if err := binary.Write(serverConn, binary.BigEndian, r.Header); err != nil {
				log.Fatalf("Error sending request header to server: %v", err)
			}
			if _, err := serverConn.Write(r.Buf); err != nil {
				log.Fatalf("Error sending request content to server: %v", err)
			}
		}

		log.Printf("reading from php-fpm")
		resps, err := readFullResponse(serverConn)
		if err != nil {
			log.Fatalf("cannot read response: %v", err)
			break
		}
		jsonRsps, err := json.Marshal(resps)
		if err != nil {
			log.Fatalf("cannot encode response to json: %v", err)
			break
		}
		log.Println("Response", string(jsonRsps))

		log.Printf("writting back to tcp client")
		for _, r := range resps {
			if err := binary.Write(clientConn, binary.BigEndian, r.Header); err != nil {
				log.Fatalf("Error sending response header to server: %v", err)
			}
			if _, err := clientConn.Write(r.Buf); err != nil {
				log.Fatalf("Error sending response content to server: %v", err)
			}
		}
	}
}

func readFullRequest(r io.Reader) ([]fcgiclient.Record, error) {
	reccords := make([]fcgiclient.Record, 0, 3)

	// recive untill empty FCGI_STDIN or EOF ?
	for {
		rec := fcgiclient.Record{}
		err := rec.Read(r)
		if err != nil && err != io.EOF {
			return nil, err
		}
		reccords = append(reccords, rec)
		if err == io.EOF {
			break
		}
		if rec.Header.Type != fcgiclient.FCGI_STDIN {
			continue
		}
		if len(rec.Content()) == 0 {
			break
		}
	}

	return reccords, nil
}

func readFullResponse(r io.Reader) ([]fcgiclient.Record, error) {
	reccords := make([]fcgiclient.Record, 0, 3)

	// recive untill EOF or FCGI_END_REQUEST
	for {
		rec := fcgiclient.Record{}
		err := rec.Read(r)
		if err != nil && err != io.EOF {
			return nil, err
		}
		reccords = append(reccords, rec)
		if err == io.EOF {
			break
		}
		if rec.Header.Type == fcgiclient.FCGI_END_REQUEST {
			break
		}
	}

	return reccords, nil
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
