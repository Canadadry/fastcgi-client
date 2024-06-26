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
	"os"
	"strconv"
)

const Action = "sniff"

type FastCGITrame struct {
	Requests  string `json:"requests"`
	Responses string `json:"responses"`
}

func Run(args []string) error {
	phpFpmAddr := "127.0.0.1:9000"
	proxyAddr := "127.0.0.1:9001"
	dontDecodeRequest := false
	help := false
	fs := flag.NewFlagSet(Action, flag.ContinueOnError)
	fs.StringVar(&phpFpmAddr, "forward-to", phpFpmAddr, "forward to fpm server at")
	fs.StringVar(&proxyAddr, "listen", proxyAddr, "proxy fastcgi listen to")
	fs.BoolVar(&dontDecodeRequest, "no-decode", dontDecodeRequest, "stop decoding request")
	fs.BoolVar(&help, "help", help, "print cmd help")
	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("cannot parse argument : %w", err)
	}
	if help {
		fs.PrintDefaults()
		return nil
	}
	l := log.New(os.Stdout, "", log.LstdFlags)
	return buildServerAndRun(
		context.Background().Done(),
		func(msg string, args ...interface{}) { l.Printf(msg, args...) },
		proxyAddr,
		phpFpmAddr,
		!dontDecodeRequest,
	)
}

type Printf func(msg string, args ...interface{})

func buildServerAndRun(done <-chan struct{}, printf Printf, proxyAddr, phpFpmAddr string, decode bool) error {
	listener, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		return fmt.Errorf("Error creating listener: %w", err)
	}
	defer listener.Close()
	printf("Proxy listening on %s, forwarding to %s", proxyAddr, phpFpmAddr)
	clientToServer := server.Pipe[[]fcgiprotocol.Record]{
		Reader: ReadFullRequest(printf),
		Writer: writeRecords,
	}
	if decode {
		clientToServer.Decoder = func(data []fcgiprotocol.Record) (interface{}, error) {
			d, err := fcgiprotocol.DecodeRequest(data)
			return d, err
		}
	}
	serverToClient := server.Pipe[[]fcgiprotocol.Record]{
		Reader:  ReadFullResponse,
		Writer:  writeRecords,
		Decoder: nil,
	}

	server.Run(
		done,
		listener,
		server.Proxy[[]fcgiprotocol.Record](
			func() (io.ReadWriteCloser, error) {
				return net.Dial("tcp", phpFpmAddr)
			},
			clientToServer,
			serverToClient,
			printf,
		),
		printf,
	)
	return nil
}

func ReadFullRequest(printf Printf) func(r io.Reader) ([]fcgiprotocol.Record, error) {
	return func(r io.Reader) ([]fcgiprotocol.Record, error) {
		reccords := make([]fcgiprotocol.Record, 0, 3)

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
			if rec.Header.Type == fcgiprotocol.FCGI_PARAMS && len(rec.Content()) == 0 {
				break
			}
		}

		req, err := fcgiprotocol.DecodeRequest(reccords)
		if err != nil {
			return reccords, fmt.Errorf("cannot decode request : %w", err)
		}
		lengthStr, _ := req.Env["CONTENT_LENGTH"]
		length, _ := strconv.Atoi(lengthStr)
		read := 0

		for read < length {
			rec := fcgiprotocol.Record{}
			err := rec.Read(r)
			if err != nil {
				return nil, err
			}
			reccords = append(reccords, rec)

			if rec.Header.Type == fcgiprotocol.FCGI_STDIN {
				read += len(rec.Content())
			}
		}

		return reccords, nil
	}
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
			hideAllEndRequestBytesButStatusCode(reccords)
			break
		}
	}

	return reccords, nil
}

func hideAllEndRequestBytesButStatusCode(reccords []fcgiprotocol.Record) {
	rec := reccords[len(reccords)-1]
	reccords[len(reccords)-1].Buf = []byte{0, 0, 0, 0, rec.Buf[4], 0, 0, 0}
}
