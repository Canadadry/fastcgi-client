package sniff

import (
	"app/fcgi/fcgiclient"
	"app/fcgi/fcgiprotocol"
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"
)

func runPhpFpmServer(t *testing.T) (string, func() error) {
	t.Helper()
	cwd, _ := os.Getwd()
	cwd = path.Join(cwd, "../../php-fpm")
	// binary can be downloaded from https://dl.static-php.dev/static-php-cli/common/
	cmd := exec.Command("./php-fpm", "-y", path.Join(cwd, "php-fpm.conf"), "-p", cwd)
	cmd.Dir = cwd
	err := cmd.Start()
	if err != nil {
		t.Fatalf("cannot start php-fpm : %v", err)
	}
	time.Sleep(time.Second)
	return cwd, cmd.Process.Kill
}

func MustUrl(t *testing.T, rawUrl string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawUrl)
	if err != nil {
		t.Fatalf("cannot parse url %s : %v", rawUrl, err)
	}
	return u
}

func MustMarshlJson(t *testing.T, data any) string {
	result, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("cannot marshal json %#v : %v", data, err)
	}
	return string(result)
}

func TestDo(t *testing.T) {
	dir, closer := runPhpFpmServer(t)
	defer func() {
		if err := closer(); err != nil {
			t.Fatalf("failed to kill process: %v", err)
		}
	}()

	tests := map[string]struct {
		In       fcgiclient.Request
		Expected fcgiclient.Response
		Error    string
		Logger   string
	}{
		"post json with body": {
			In: fcgiclient.Request{
				ID:           1,
				DocumentRoot: dir,
				Method:       "POST",
				Url:          MustUrl(t, "/api/auth-tokens?status_code=201"),
				Body:         `{"login":admin","password":"azertyu"}` + "\n",
				Index:        "index.php",
				Env:          map[string]string{},
				Header: map[string]string{
					"Content-type": "application/json",
				},
			},
			Expected: fcgiclient.Response{
				StatusCode: 201,
				Header: map[string]string{
					"Content-type":  "text/html; charset=UTF-8",
					"X-Powered-By":  "PHP/8.3.7",
					"X-Request-Uri": "/api/auth-tokens?status_code=201",
					"Status":        "201 Created",
					"X-Status-Code": "201",
					"Status-Code":   "201",
				},
				Stdout: strings.Join([]string{
					"<h1>Requested URL:</h1>",
					"<p>/api/auth-tokens?status_code=201</p>",
					"<h1>Request Method:</h1>",
					"<p>POST</p>",
					"<h1>Headers:</h1>",
					"<pre>",
					"Content-Length: 38",
					"Content-Type: application/json",
					"</pre>",
					"<h1>Body:</h1>",
					"<pre>",
					`{"login":admin","password":"azertyu"}`,
					"</pre>",
				}, "\n"),
				Stderr: "",
			},
			Logger: strings.Join([]string{
				"Proxy listening on 127.0.0.1:9001, forwarding to 127.0.0.1:9000",
				"handling new TCP client",
				"connected to server",
				"request read raw " + MustMarshlJson(t, []fcgiprotocol.Record{
					buildRecord(fcgiprotocol.FCGI_BEGIN_REQUEST, []byte{0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}),
					pairRecord(t, map[string]string{
						"CONTENT_LENGTH":    "38",
						"CONTENT_TYPE":      "application/json",
						"DOCUMENT_ROOT":     dir,
						"DOCUMENT_URI":      "/api/auth-tokens",
						"GATEWAY_INTERFACE": "CGI/1.1",
						"HTTP_CONTENT_TYPE": "application/json",
						"QUERY_STRING":      "status_code=201",
						"REQUEST_METHOD":    "POST",
						"REQUEST_SCHEME":    "http",
						"REQUEST_URI":       "/api/auth-tokens?status_code=201",
						"SCRIPT_FILENAME":   path.Join(dir, "index.php"),
						"SCRIPT_NAME":       "/api/auth-tokens",
						"SERVER_PROTOCOL":   "HTTP/1.1",
						"SERVER_SOFTWARE":   "go / fcgiclient ",
					}),
					buildRecord(fcgiprotocol.FCGI_PARAMS, []byte{}),
					buildRecord(fcgiprotocol.FCGI_STDIN, []byte(`{"login":admin","password":"azertyu"}`+"\n")),
				}),
				"writing back request",
				"finish writing to server, waiting for response",
				"response read raw " + MustMarshlJson(t, []fcgiprotocol.Record{
					buildRecord(fcgiprotocol.FCGI_STDOUT, []byte(strings.Join([]string{
						"Status: 201 Created",
						"X-Powered-By: PHP/8.3.7",
						"Status-Code:201",
						"X-Status-Code: 201",
						"X-Request-Uri: /api/auth-tokens?status_code=201",
						"Content-type: text/html; charset=UTF-8",
						"",
						""}, "\r\n",
					)+strings.Join([]string{
						"<h1>Requested URL:</h1>",
						"<p>/api/auth-tokens?status_code=201</p>",
						"<h1>Request Method:</h1>",
						"<p>POST</p>",
						"<h1>Headers:</h1>",
						"<pre>",
						"Content-Length: 38",
						"Content-Type: application/json",
						"</pre>",
						"<h1>Body:</h1>",
						"<pre>",
						`{"login":admin","password":"azertyu"}`,
						"</pre>",
					}, "\n"))),
					buildRecord(fcgiprotocol.FCGI_END_REQUEST, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}),
				}),
				"writing back response",
				"",
			}, "\n"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			done := make(chan struct{})
			defer close(done)
			buf := &bytes.Buffer{}
			l := log.New(buf, "", 0)
			printf := func(msg string, args ...interface{}) {
				l.Printf(msg, args...)
				t.Logf(msg, args...)
			}
			go buildServerAndRun(done, printf, "127.0.0.1:9001", "127.0.0.1:9000", false)
			time.Sleep(time.Second)
			conn, err := net.Dial("tcp", "127.0.0.1:9001")
			if err != nil {
				t.Fatalf("cannot dial php server : %v", err)
			}
			defer conn.Close()
			result, err := fcgiclient.Do(conn, tt.In)
			testError(t, err, tt.Error)
			if !reflect.DeepEqual(tt.Expected, result) {
				t.Fatalf("want \n%#v\ngot \n%#v\n", tt.Expected, result)
			}
			if buf.String() != tt.Logger {
				t.Fatalf("want \n%#v\ngot \n%#v\n", tt.Logger, buf.String())
			}
		})
	}
}

func testError(t *testing.T, got error, want string) {
	t.Helper()
	if got != nil {
		if want == "" {
			t.Fatalf("failed running request : %v", got)
		} else {
			if got.Error() != want {
				t.Fatalf("expected error want '%s' got '%s'", want, got.Error())
			}
		}
	} else {
		if want != "" {
			if got.Error() != want {
				t.Fatalf("expected error %s got nil", want)
			}
		}
	}
}

func pairRecord(t *testing.T, pairs map[string]string) fcgiprotocol.Record {
	t.Helper()
	buf := &bytes.Buffer{}
	err := fcgiprotocol.BuildPair(buf, pairs)
	if err != nil {
		t.Fatalf("cannot build pair :%v", err)
	}
	return buildRecord(fcgiprotocol.FCGI_PARAMS, buf.Bytes())
}

func buildRecord(recType uint8, data []byte) fcgiprotocol.Record {
	rec := fcgiprotocol.Record{
		Header: fcgiprotocol.NewHeader(recType, 1, len(data)),
	}
	for _ = range rec.Header.PaddingLength {
		data = append(data, 0)
	}
	rec.Buf = data
	return rec
}
