package client

import (
	"app/decoder"
	"app/fcgiclient"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

const Action = "client"

type FCGIRequest struct {
	Method       string
	Url          *url.URL
	Body         string
	Index        string
	DocumentRoot string
	Env          map[string]string
	Header       map[string]string
}

func Do(host string, req FCGIRequest) error {
	env := map[string]string{
		"CONTENT_LENGTH":    fmt.Sprintf("%d", len(req.Body)),
		"CONTENT_TYPE":      http.DetectContentType([]byte(req.Body[:min(len(req.Body), 512)])),
		"DOCUMENT_URI":      req.Url.Path,
		"GATEWAY_INTERFACE": "CGI/1.1",
		"REQUEST_SCHEME":    "http",
		"SERVER_PROTOCOL":   "HTTP/1.1",
		"REQUEST_METHOD":    req.Method,
		"SCRIPT_FILENAME":   path.Join(req.DocumentRoot + req.Index),
		"SCRIPT_NAME":       req.Url.Path,
		"SERVER_SOFTWARE":   "go / fcgiclient ",
		"DOCUMENT_ROOT":     req.DocumentRoot,
		"QUERY_STRING":      req.Url.RawQuery,
		"REQUEST_URI":       req.Url.Path,
	}

	for header, values := range req.Header {
		env["HTTP_"+strings.Replace(strings.ToUpper(header), "-", "_", -1)] = values
	}

	for name, value := range req.Env {
		env[name] = value
	}

	conn, err := net.Dial("tcp", host)
	if err != nil {
		return fmt.Errorf("cannot open conn to php server: %w", err)
	}
	fcgi := fcgiclient.New(conn)

	content, stderr, err := fcgi.Request(env, req.Body)

	if err != nil {
		return fmt.Errorf("cannot send fcgi request: %w : %s", err, string(stderr))
	}

	rsp, err := decoder.ParseResponse(fmt.Sprintf("%s", content))
	if err != nil {
		return fmt.Errorf("cannot read fcgi reqponse: %w : %s", err, string(stderr))
	}

	fmt.Println("statusCode", rsp.StatusCode, "headers", rsp.Headers, "body", rsp.Stdout, "stderr", string(stderr))
	return nil
}

func ParseFastCgiResponse(content string) (int, map[string]string, string, error) {
	var headers map[string]string

	parts := strings.SplitN(content, "\r\n\r\n", 2)

	if len(parts) < 2 {
		return 502, headers, "", fmt.Errorf("Cannot parse FastCGI Response expect two part got %v \n -%s-", len(parts), content)
	}

	headerParts := strings.Split(parts[0], ":")
	body := parts[1]
	status := 200

	if strings.HasPrefix(headerParts[0], "Status:") {
		lineParts := strings.SplitN(headerParts[0], " ", 3)
		status, _ = strconv.Atoi(lineParts[1])
	}

	for _, line := range headerParts {
		lineParts := strings.SplitN(line, ":", 2)

		if len(lineParts) < 2 {
			continue
		}

		lineParts[1] = strings.TrimSpace(lineParts[1])

		if lineParts[0] == "Status" {
			continue
		}

		headers[lineParts[0]] = lineParts[1]
	}

	return status, headers, body, nil
}

func Run(args []string) error {

	cwd, _ := os.Getwd()
	rawUrl := "/"
	req := FCGIRequest{
		Method:       "GET",
		DocumentRoot: cwd,
		Index:        "index.php",
	}
	host := "127.0.0.1:9000"
	env := ""
	header := "{}"
	help := false
	fs := flag.NewFlagSet(Action, flag.ContinueOnError)
	fs.StringVar(&host, "host", host, "php-fmp hostname")
	fs.StringVar(&req.Method, "method", req.Method, "request method")
	fs.StringVar(&rawUrl, "url", rawUrl, "request url")
	fs.StringVar(&req.Index, "index", req.Index, "request index")
	fs.StringVar(&req.DocumentRoot, "document-root", req.DocumentRoot, "request document root")
	fs.StringVar(&req.Body, "body", req.Body, "request body")
	fs.StringVar(&env, "env", env, "request env as json")
	fs.StringVar(&header, "header", header, "request header as json")
	fs.BoolVar(&help, "help", help, "print cmd help")
	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("cannot parse argument : %w", err)
	}
	if help {
		fs.PrintDefaults()
		return nil
	}
	if env != "" {
		f, err := os.Open(env)
		if err != nil {
			return fmt.Errorf("cannot open env file : %w", err)
		}
		defer f.Close()
		err = json.NewDecoder(f).Decode(&req.Env)
		if err != nil {
			return fmt.Errorf("cannot read env json data : %w", err)
		}
	}

	err = json.Unmarshal([]byte(header), &req.Header)
	if err != nil {
		return fmt.Errorf("cannot read header json data : %w", err)
	}

	req.Url, err = url.Parse(rawUrl)
	if err != nil {
		return fmt.Errorf("cannot parse input url : %w", err)
	}

	return Do(host, req)
}
