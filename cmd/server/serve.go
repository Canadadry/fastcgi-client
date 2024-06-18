// forked from https://github.com/beberlei/fastcgi-serv

package server

import (
	"app/decoder"
	"app/fcgiclient"
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

const Action = "server"

var documentRoot string
var index string
var listen string
var staticHandler *http.ServeMux
var server string
var serverEnvironment map[string]string

func respond(w http.ResponseWriter, body string, statusCode int, headers map[string]string) {
	w.WriteHeader(statusCode)
	for header, value := range headers {
		w.Header().Set(header, value)
	}
	fmt.Fprintf(w, "%s", body)
}

func handler(w http.ResponseWriter, r *http.Request) {
	var filename string
	var scriptName string

	rBody, _ := io.ReadAll(r.Body)

	if r.URL.Path == "/.env" {
		respond(w, "Not allowed", 403, map[string]string{})
		return
	} else if r.URL.Path == "/" || r.URL.Path == "" {
		scriptName = "/" + index
		filename = documentRoot + "/" + index
	} else {
		scriptName = r.URL.Path
		filename = documentRoot + r.URL.Path
	}

	// static file exists
	_, err := os.Stat(filename)
	if !strings.HasSuffix(filename, ".php") && err == nil {
		staticHandler.ServeHTTP(w, r)
		return
	}

	if os.IsNotExist(err) {
		scriptName = "/" + index
		filename = documentRoot + "/" + index
	}

	env := make(map[string]string)

	for name, value := range serverEnvironment {
		env[name] = value
	}

	env["REQUEST_METHOD"] = r.Method
	env["SCRIPT_FILENAME"] = filename
	env["SCRIPT_NAME"] = scriptName
	env["SERVER_SOFTWARE"] = "go / fcgiclient "
	env["REMOTE_ADDR"] = r.RemoteAddr
	env["SERVER_PROTOCOL"] = "HTTP/1.1"
	env["PATH_INFO"] = r.URL.Path
	env["DOCUMENT_ROOT"] = documentRoot
	env["QUERY_STRING"] = r.URL.RawQuery
	env["REQUEST_URI"] = r.URL.Path + "?" + r.URL.RawQuery
	//env["HTTP_HOST"] = r.Host
	//env["SERVER_ADDR"] = listen

	for header, values := range r.Header {
		env["HTTP_"+strings.Replace(strings.ToUpper(header), "-", "_", -1)] = values[0]
	}

	conn, err := net.Dial("tcp", server)
	if err != nil {
		fmt.Printf("err: %v", err)
	}

	fcgi := fcgiclient.New(conn)
	content, stderr, err := fcgi.Request(env, string(rBody))

	if err != nil {
		fmt.Printf("ERROR: %s - %v", r.URL.Path, err)
	}

	rsp, err := decoder.ParseResponse(fmt.Sprintf("%s", content))

	respond(w, rsp.Stdout, rsp.StatusCode, rsp.Headers)

	fmt.Printf("%s \"%s %s %s\" %d %d \"%s\" \"%s\"\n", r.RemoteAddr, r.Method, r.URL.Path, r.Proto, rsp.StatusCode, len(content), r.UserAgent(), stderr)
}

func ReadEnvironmentFile(path string) {
	file, err := os.Open(path + "/.env")

	if err != nil {
		return
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	serverEnvironment = make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			serverEnvironment[parts[0]] = parts[1]
		}
	}
}

func Run(args []string) error {

	cwd, _ := os.Getwd()
	fs := flag.NewFlagSet(Action, flag.ContinueOnError)
	fs.StringVar(&documentRoot, "document-root", cwd, "The document root to serve files from")
	fs.StringVar(&listen, "listen", "localhost:8080", "The webserver bind address to listen to.")
	fs.StringVar(&server, "server", "127.0.0.1:9000", "The FastCGI Server to listen to")
	fs.StringVar(&index, "index", "index.php", "The default script to call when path cannot be served by existing file.")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("cannot parse argument : %w", err)
	}

	ReadEnvironmentFile(cwd)

	staticHandler = http.NewServeMux()
	staticHandler.Handle("/", http.FileServer(http.Dir(documentRoot)))

	fmt.Printf("Listening on http://%s\n", listen)
	fmt.Printf("Document root is %s\n", documentRoot)
	fmt.Printf("Press Ctrl-C to quit.\n")

	http.HandleFunc("/", handler)
	http.ListenAndServe(listen, nil)
	return nil
}
