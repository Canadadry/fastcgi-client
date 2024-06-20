package fcgiclient

import (
	"app/fcgi/fcgiprotocol"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type Request struct {
	Method       string
	Url          *url.URL
	Body         string
	Index        string
	DocumentRoot string
	Env          map[string]string
	Header       map[string]string
}

type Response struct {
	StatusCode int
	Header     map[string]string
	Stdout     string
	Stderr     string
}

func Do(rw io.ReadWriter, req Request) (Response, error) {

	env := map[string]string{
		"CONTENT_LENGTH":    fmt.Sprintf("%d", len(req.Body)),
		"CONTENT_TYPE":      http.DetectContentType([]byte(req.Body[:min(len(req.Body), 512)])),
		"DOCUMENT_URI":      req.Url.Path,
		"GATEWAY_INTERFACE": "CGI/1.1",
		"REQUEST_SCHEME":    "http",
		"SERVER_PROTOCOL":   "HTTP/1.1",
		"REQUEST_METHOD":    req.Method,
		"SCRIPT_FILENAME":   path.Join(req.DocumentRoot, req.Index),
		"SCRIPT_NAME":       req.Url.Path,
		"SERVER_SOFTWARE":   "go / fcgiclient ",
		"DOCUMENT_ROOT":     req.DocumentRoot,
		"QUERY_STRING":      req.Url.RawQuery,
		"REQUEST_URI":       req.Url.Path,
	}

	for header, values := range req.Header {
		env["HTTP_"+strings.Replace(strings.ToUpper(header), "-", "_", -1)] = values
	}

	if ct, ok := env["HTTP_CONTENT_TYPE"]; ok {
		env["CONTENT_TYPE"] = ct
	}

	for name, value := range req.Env {
		env[name] = value
	}

	fcgi := fcgiprotocol.New(rw)

	content, stderr, err := fcgi.Request(env, req.Body)

	if err != nil {
		return Response{}, fmt.Errorf("cannot send fcgi request: %w : %s", err, string(stderr))
	}

	rsp, err := fcgiprotocol.ParseResponse(fmt.Sprintf("%s", content))
	if err != nil {
		return Response{}, fmt.Errorf("cannot read fcgi reqponse: %w : %s", err, string(stderr))
	}

	return Response{
		StatusCode: rsp.StatusCode,
		Header:     rsp.Headers,
		Stdout:     rsp.Stdout,
		Stderr:     string(stderr),
	}, nil
}
