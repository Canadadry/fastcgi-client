package fcgiclient

import (
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
		In       Request
		Expected Response
	}{
		"main script not found": {
			In: Request{
				DocumentRoot: dir,
				Method:       "GET",
				Url:          MustUrl(t, "/"),
				Body:         "",
				Index:        "wrong.php",
				Env:          map[string]string{},
				Header:       map[string]string{},
			},
			Expected: Response{
				StatusCode: 404,
				Header: map[string]string{
					"Content-type": "text/html; charset=UTF-8",
					"X-Powered-By": "PHP/8.3.7",
					"Status":       "404 Not Found",
				},
				Stdout: "File not found.\n",
				Stderr: "Primary script unknown",
			},
		},
		"basic": {
			In: Request{
				DocumentRoot: dir,
				Method:       "GET",
				Url:          MustUrl(t, "/"),
				Body:         "",
				Index:        "index.php",
				Env:          map[string]string{},
				Header:       map[string]string{},
			},
			Expected: Response{
				StatusCode: 200,
				Header: map[string]string{
					"Content-type":  "text/html; charset=UTF-8",
					"X-Powered-By":  "PHP/8.3.7",
					"X-Request-Uri": "/",
				},
				Stdout: strings.Join([]string{
					"<h1>Requested URL:</h1>",
					"<p>/</p>",
					"<h1>Request Method:</h1>",
					"<p>GET</p>",
					"<h1>Headers:</h1>",
					"<pre>",
					"Content-Length: 0",
					"Content-Type: text/plain; charset=utf-8",
					"</pre>",
					"<h1>Body:</h1>",
					"<pre>",
					"</pre>",
				}, "\n"),
				Stderr: "",
			},
		},
		"basic with status code": {
			In: Request{
				DocumentRoot: dir,
				Method:       "GET",
				Url:          MustUrl(t, "/?status_code=403"),
				Body:         "",
				Index:        "index.php",
				Env:          map[string]string{},
				Header:       map[string]string{},
			},
			Expected: Response{
				StatusCode: 403,
				Header: map[string]string{
					"Content-type":  "text/html; charset=UTF-8",
					"X-Powered-By":  "PHP/8.3.7",
					"X-Request-Uri": "/",
					"Status":        "403 Forbidden",
					"X-Status-Code": "403",
					"Status-Code":   "403",
				},
				Stdout: strings.Join([]string{
					"<h1>Requested URL:</h1>",
					"<p>/</p>",
					"<h1>Request Method:</h1>",
					"<p>GET</p>",
					"<h1>Headers:</h1>",
					"<pre>",
					"Content-Length: 0",
					"Content-Type: text/plain; charset=utf-8",
					"</pre>",
					"<h1>Body:</h1>",
					"<pre>",
					"</pre>",
				}, "\n"),
				Stderr: "",
			},
		},
		"option cors request": {
			In: Request{
				DocumentRoot: dir,
				Method:       "OPTIONS",
				Url:          MustUrl(t, "/api/users"),
				Body:         "",
				Index:        "index.php",
				Env:          map[string]string{},
				Header: map[string]string{
					"Access-Control-Request-Method":  "POST",
					"Access-Control-Request-Headers": "content-type",
					"Referer":                        "https://verification.exemple.com/",
					"Origin":                         "https://verification.exemple.com/",
				},
			},
			Expected: Response{
				StatusCode: 200,
				Header: map[string]string{
					"Content-type":  "text/html; charset=UTF-8",
					"X-Powered-By":  "PHP/8.3.7",
					"X-Request-Uri": "/api/users",
				},
				Stdout: strings.Join([]string{
					"<h1>Requested URL:</h1>",
					"<p>/api/users</p>",
					"<h1>Request Method:</h1>",
					"<p>OPTIONS</p>",
					"<h1>Headers:</h1>",
					"<pre>",
					"Access-Control-Request-Headers: content-type",
					"Access-Control-Request-Method: POST",
					"Content-Length: 0",
					"Content-Type: text/plain; charset=utf-8",
					"Origin: https://verification.exemple.com/",
					"Referer: https://verification.exemple.com/",
					"</pre>",
					"<h1>Body:</h1>",
					"<pre>",
					"</pre>",
				}, "\n"),
				Stderr: "",
			},
		},
		"post json with body": {
			In: Request{
				DocumentRoot: dir,
				Method:       "POST",
				Url:          MustUrl(t, "/api/auth-tokens"),
				Body:         `{"login":admin","password":"azertyu"}` + "\n",
				Index:        "index.php",
				Env:          map[string]string{},
				Header: map[string]string{
					"Content-type": "application/json",
				},
			},
			Expected: Response{
				StatusCode: 200,
				Header: map[string]string{
					"Content-type":  "text/html; charset=UTF-8",
					"X-Powered-By":  "PHP/8.3.7",
					"X-Request-Uri": "/api/auth-tokens",
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
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			conn, err := net.Dial("tcp", "127.0.0.1:9000")
			if err != nil {
				t.Fatalf("cannot dial php server : %v", err)
			}
			defer conn.Close()
			result, err := Do(conn, tt.In)
			if err != nil {
				t.Fatalf("failed running request : %v", err)
			}
			if !reflect.DeepEqual(tt.Expected, result) {
				t.Fatalf("want \n%#v\ngot \n%#v\n", tt.Expected, result)
			}
		})
	}

}
