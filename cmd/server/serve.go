package server

import (
	"app/fcgi/fcgiclient"
	"app/pkg/http/handler"
	"app/pkg/http/middleware"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
)

const Action = "server"

func Run(args []string) error {

	cwd, _ := os.Getwd()
	listen := "localhost:8080"
	srv := Server{
		DocumentRoot: cwd,
		FCGIHost:     "127.0.0.1:9000",
		Index:        "index.php",
		IP:           "127.0.0.1",
		Name:         "localhost",
		Port:         "443",
	}
	fs := flag.NewFlagSet(Action, flag.ContinueOnError)
	fs.StringVar(&srv.DocumentRoot, "document-root", srv.DocumentRoot, "The document root to serve files from")
	fs.StringVar(&listen, "listen", listen, "The webserver bind address to listen to.")
	fs.StringVar(&srv.IP, "srv-ip", srv.IP, "The webserver ip passed to php-fpm.")
	fs.StringVar(&srv.Name, "srv-name", srv.Name, "The webserver name passed to php-fpm.")
	fs.StringVar(&srv.Port, "srv-port", srv.Port, "The webserver port passed to php-fpm.")
	fs.StringVar(&srv.FCGIHost, "server", srv.FCGIHost, "The FastCGI Server to listen to")
	fs.StringVar(&srv.Index, "index", srv.Index, "The default script to call when path cannot be served by existing file.")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("cannot parse argument : %w", err)
	}

	fmt.Printf("Listening on http://%s\n", listen)
	fmt.Printf("Document root is %s\n", srv.DocumentRoot)
	fmt.Printf("Press Ctrl-C to quit.\n")

	http.HandleFunc("/", middleware.HandleWithLogAndError(handle(srv)))
	http.ListenAndServe(listen, nil)
	return nil
}

type Server struct {
	DocumentRoot string
	Index        string
	FCGIHost     string
	IP           string
	Name         string
	Port         string
}

func handle(srv Server) func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	sh := handler.Static(srv.DocumentRoot, srv.Index)
	fh := fcgiHandler(srv)
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		if ok := sh(w, r); ok {
			return nil, nil
		}
		return fh(w, r)
	}
}

func fcgiHandler(srv Server) func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		defer r.Body.Close()
		rBody, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read request body: %v", err)
		}

		remoteAddr, remotePort := splitIPAndPort(r.RemoteAddr)

		req := fcgiclient.Request{
			DocumentRoot: srv.DocumentRoot,
			Index:        srv.Index,
			Method:       r.Method,
			Url:          r.URL,
			Body:         string(rBody),
			Header:       map[string]string{},
			Env: map[string]string{
				"REMOTE_ADDR": remoteAddr,
				"REMOTE_PORT": remotePort,
				"SERVER_ADDR": srv.IP,
				"SERVER_NAME": srv.Name,
				"SERVER_PORT": srv.Port,
			},
		}

		for name, values := range r.Header {
			req.Header[name] = values[0]
		}

		conn, err := net.Dial("tcp", srv.FCGIHost)
		if err != nil {
			return nil, fmt.Errorf("cannot dial php server : %w", err)
		}
		defer conn.Close()

		rsp, err := fcgiclient.Do(conn, req)
		if err != nil {
			return nil, fmt.Errorf("cannot make request to php : %v", err)
		}

		middleware.Respond(w, rsp.Stdout, rsp.StatusCode, rsp.Header)

		return []byte(rsp.Stderr), nil
	}
}
