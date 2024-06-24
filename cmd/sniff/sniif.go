package sniff

import (
	"app/fcgi/fcgiprotocol"
	"app/pkg/server"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
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
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	log.Printf("Proxy listening on %s, forwarding to %s", proxyAddr, phpFpmAddr)
	clientToServer := server.Pipe[[]fcgiprotocol.Record]{
		Reader: ReadFullRequest,
		Writer: writeRecords,
		Decoder: func(data []fcgiprotocol.Record) (interface{}, error) {
			d, err := fcgiprotocol.DecodeRequest(data)
			return d, err
		},
	}
	serverToClient := server.Pipe[[]fcgiprotocol.Record]{
		Reader:  ReadFullResponse,
		Writer:  writeRecords,
		Decoder: nil,
	}

	server.Run(
		context.Background().Done(),
		listener,
		server.Proxy[[]fcgiprotocol.Record](
			func() (io.ReadWriteCloser, error) {
				return net.Dial("tcp", phpFpmAddr)
			},
			clientToServer,
			serverToClient,
		),
	)
	return nil
}

func ReadFullRequest(r io.Reader) ([]fcgiprotocol.Record, error) {
	reccords := make([]fcgiprotocol.Record, 0, 3)

	// recive untill empty FCGI_STDIN or EOF ?
	for {
		rec := fcgiprotocol.Record{}
		err := rec.Read(r)
		if err != nil && err != io.EOF {
			return nil, err
		}
		reccords = append(reccords, rec)
		if err == io.EOF {
			break
		}
		if rec.Header.Type == fcgiprotocol.FCGI_STDIN && len(rec.Content()) == 0 {
			break
		}
		if rec.Header.Type == fcgiprotocol.FCGI_PARAMS && len(rec.Content()) == 0 {
			break
		}
	}

	return reccords, nil
}

func writeRecords(w io.Writer, recs []fcgiprotocol.Record) error {
	for _, r := range recs {
		if err := binary.Write(w, binary.BigEndian, r.Header); err != nil {
			return fmt.Errorf("Error sending request header to server: %w", err)
		}
		if _, err := w.Write(r.Buf); err != nil {
			return fmt.Errorf("Error sending request content to server: %w", err)
		}
	}
	return nil
}

func ReadFullResponse(r io.Reader) ([]fcgiprotocol.Record, error) {
	reccords := make([]fcgiprotocol.Record, 0, 3)

	// recive untill EOF or FCGI_END_REQUEST
	for {
		rec := fcgiprotocol.Record{}
		err := rec.Read(r)
		if err != nil && err != io.EOF {
			return nil, err
		}
		reccords = append(reccords, rec)
		if err == io.EOF {
			break
		}
		if rec.Header.Type == fcgiprotocol.FCGI_END_REQUEST {
			break
		}
	}

	return reccords, nil
}
