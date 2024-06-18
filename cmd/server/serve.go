// forked from https://github.com/beberlei/fastcgi-serv

package server

import (
	"app/decoder"
	"app/fcgiclient"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

const Action = "server"

type Server struct {
	DocumentRoot  string
	Index         string
	StaticHandler *http.ServeMux
	FCGIHost      string
}

func respond(w http.ResponseWriter, body string, statusCode int, headers map[string]string) {
	w.WriteHeader(statusCode)
	for header, value := range headers {
		w.Header().Set(header, value)
	}
	fmt.Fprintf(w, "%s", body)
}

func handler(srv Server) func(w http.ResponseWriter, r *http.Request) {
	staticHandler := func(w http.ResponseWriter, r *http.Request) bool {
		filename := srv.DocumentRoot + r.URL.Path
		if r.URL.Path == "/.env" || r.URL.Path == "/" || r.URL.Path == "" {
			filename = srv.DocumentRoot + "/" + srv.Index
		}
		_, err := os.Stat(filename)
		if err == nil && !strings.HasSuffix(filename, ".php") {
			srv.StaticHandler.ServeHTTP(w, r)
			return true
		}
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("cannot stat %s : %v\n", filename, err)
			respond(w, "server error", 500, nil)
			return true
		}
		return false
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if staticHandler(w, r) {
			return
		}
		rBody, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("cannot read request body: %v", err)
			respond(w, "server error", 500, nil)
			return
		}

		env := map[string]string{
			"REQUEST_METHOD":  r.Method,
			"SCRIPT_FILENAME": srv.DocumentRoot + "/" + srv.Index,
			"SCRIPT_NAME":     "/" + srv.Index,
			"SERVER_SOFTWARE": "go / fcgiclient ",
			"REMOTE_ADDR":     r.RemoteAddr,
			"SERVER_PROTOCOL": "HTTP/1.1",
			"PATH_INFO":       r.URL.Path,
			"DOCUMENT_ROOT":   srv.DocumentRoot,
			"QUERY_STRING":    r.URL.RawQuery,
			"REQUEST_URI":     r.URL.Path + "?" + r.URL.RawQuery,
			//env["HTTP_HOST"] = r.Host
			//env["SERVER_ADDR"] = listen
		}

		for header, values := range r.Header {
			env["HTTP_"+strings.Replace(strings.ToUpper(header), "-", "_", -1)] = values[0]
		}

		conn, err := net.Dial("tcp", srv.FCGIHost)
		if err != nil {
			fmt.Printf("cannot read request body: %v", err)
			respond(w, "cannot found server", 502, nil)
			return
		}

		fcgi := fcgiclient.New(conn)
		content, stderr, err := fcgi.Request(env, string(rBody))
		if err != nil {
			fmt.Printf("while request php-fpm %s : %v", r.URL.Path, err)
			respond(w, "server error", 500, nil)
			return
		}

		rsp, err := decoder.ParseResponse(fmt.Sprintf("%s", content))
		if err != nil {
			fmt.Printf("cannot decode response of %s : %v", r.URL.Path, err)
			respond(w, "server error", 500, nil)
			return
		}

		respond(w, rsp.Stdout, rsp.StatusCode, rsp.Headers)

		fmt.Printf("%s \"%s %s %s\" %d %d \"%s\" \"%s\"\n", r.RemoteAddr, r.Method, r.URL.Path, r.Proto, rsp.StatusCode, len(content), r.UserAgent(), stderr)
	}
}

func Run(args []string) error {

	cwd, _ := os.Getwd()
	listen := "localhost:8080"
	srv := Server{
		DocumentRoot: cwd,
		FCGIHost:     "127.0.0.1:9000",
		Index:        "index.php",
	}
	fs := flag.NewFlagSet(Action, flag.ContinueOnError)
	fs.StringVar(&srv.DocumentRoot, "document-root", srv.DocumentRoot, "The document root to serve files from")
	fs.StringVar(&listen, "listen", listen, "The webserver bind address to listen to.")
	fs.StringVar(&srv.FCGIHost, "server", srv.FCGIHost, "The FastCGI Server to listen to")
	fs.StringVar(&srv.Index, "index", srv.Index, "The default script to call when path cannot be served by existing file.")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("cannot parse argument : %w", err)
	}

	srv.StaticHandler = http.NewServeMux()
	srv.StaticHandler.Handle("/", http.FileServer(http.Dir(srv.DocumentRoot)))

	fmt.Printf("Listening on http://%s\n", listen)
	fmt.Printf("Document root is %s\n", srv.DocumentRoot)
	fmt.Printf("Press Ctrl-C to quit.\n")

	http.HandleFunc("/", handler(srv))
	http.ListenAndServe(listen, nil)
	return nil
}
