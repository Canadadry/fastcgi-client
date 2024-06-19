// forked from https://github.com/beberlei/fastcgi-serv

package server

import (
	"app/decoder"
	"app/fcgiclient"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const Action = "server"

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

	http.HandleFunc("/", HandleWithLogAndError(handler(srv)))
	http.ListenAndServe(listen, nil)
	return nil
}

type Server struct {
	DocumentRoot  string
	Index         string
	StaticHandler *http.ServeMux
	FCGIHost      string
}

func respond(w http.ResponseWriter, body string, statusCode int, headers map[string]string) {
	for header, value := range headers {
		w.Header().Set(header, value)
	}
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, "%s", body)
}

func HandleWithLogAndError(next func(w http.ResponseWriter, r *http.Request) ([]byte, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		ww := &wrapWriter{w: rw, statusCode: http.StatusOK}
		startedAt := time.Now()
		stderr, err := next(ww, r)
		endedAt := time.Now()
		logLine := struct {
			At           time.Time
			Level        string
			Method       string
			URL          string
			Status       int
			Bytes        int
			Elapsed      string
			ErrorMessage string
			Stderr       []byte
		}{
			At:           startedAt,
			Level:        "trace",
			Method:       r.Method,
			URL:          r.URL.String(),
			Status:       ww.statusCode,
			Bytes:        ww.byteWritten,
			Elapsed:      fmt.Sprintf("%v", endedAt.Sub(startedAt)),
			ErrorMessage: "",
			Stderr:       stderr,
		}
		if len(stderr) > 0 {
			logLine.Level = "error"
		}
		if logLine.Status >= 500 {
			logLine.Level = "error"
		}
		if err != nil {
			logLine.Level = "error"
			logLine.ErrorMessage = err.Error()
			respond(rw, "server error", 500, nil)
		}
		_ = json.NewEncoder(os.Stdout).Encode(logLine)
	}
}

func handler(srv Server) func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
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
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		defer r.Body.Close()
		if staticHandler(w, r) {
			return nil, nil
		}
		rBody, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read request body: %v", err)
		}

		env := map[string]string{
			"CONTENT_LENGTH":    fmt.Sprintf("%d", len(rBody)),
			"CONTENT_TYPE":      http.DetectContentType([]byte(rBody[:min(len(rBody), 512)])),
			"DOCUMENT_URI":      r.URL.Path,
			"GATEWAY_INTERFACE": "CGI/1.1",
			"REQUEST_SCHEME":    "http",
			"SERVER_PROTOCOL":   "HTTP/1.1",
			"REQUEST_METHOD":    r.Method,
			"SCRIPT_FILENAME":   srv.DocumentRoot + "/" + srv.Index,
			"SCRIPT_NAME":       r.URL.Path,
			"SERVER_SOFTWARE":   "go / fcgiclient ",
			"DOCUMENT_ROOT":     srv.DocumentRoot,
			"QUERY_STRING":      r.URL.RawQuery,
			"REQUEST_URI":       r.URL.Path,
		}

		for header, values := range r.Header {
			name := "HTTP_" + strings.Replace(strings.ToUpper(header), "-", "_", -1)
			env[name] = values[0]
		}

		conn, err := net.Dial("tcp", srv.FCGIHost)
		if err != nil {
			return nil, fmt.Errorf("cannot read request body: %w", err)
		}
		defer conn.Close()

		fcgi := fcgiclient.New(conn)
		content, stderr, err := fcgi.Request(env, string(rBody))
		if err != nil {
			return stderr, fmt.Errorf("while request php-fpm %s : %w", r.URL.Path, err)
		}

		rsp, err := decoder.ParseResponse(fmt.Sprintf("%s", content))
		if err != nil {
			return stderr, fmt.Errorf("cannot decode response of %s : %w", r.URL.Path, err)
		}

		respond(w, rsp.Stdout, rsp.StatusCode, rsp.Headers)

		return stderr, nil
	}
}

type wrapWriter struct {
	w           http.ResponseWriter
	statusCode  int
	byteWritten int
}

func (ww *wrapWriter) Header() http.Header {
	return ww.w.Header()
}
func (ww *wrapWriter) Write(b []byte) (int, error) {
	n, err := ww.w.Write(b)
	ww.byteWritten += n
	return n, err
}
func (ww *wrapWriter) WriteHeader(statusCode int) {
	ww.statusCode = statusCode
	ww.w.WriteHeader(statusCode)
}
