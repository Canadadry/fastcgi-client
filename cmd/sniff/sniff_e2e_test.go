package sniff

import (
	"app/fcgi/fcgiclient"
	"bytes"
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
	t.Logf("dir is %s", cwd)
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
					"X-Request-Uri": "/api/auth-tokens",
					"Status":        "201 Created",
					"X-Status-Code": "201",
					"Status-Code":   "201",
				},
				Stdout: strings.Join([]string{
					"<h1>Requested URL:</h1>",
					"<p>/api/auth-tokens</p>",
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
			Logger: strings.Join([]string{"test"}, "\n"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			done := make(chan struct{})
			defer close(done)
			buf := &bytes.Buffer{}
			go buildServerAndRun(done, log.New(buf, "", 0), "127.0.0.1:9001", "127.0.0.1:9000")
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
