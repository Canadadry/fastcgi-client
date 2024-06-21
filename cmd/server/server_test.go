package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
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

type Request struct {
	Method string
	URL    string
	Body   string
	Header map[string]string
}

type Response struct {
	StatusCode int
	Body       string
	Header     map[string]string
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
		Error    string
	}{
		"basic": {
			In: Request{
				Method: "POST",
				URL:    "/test?status_code=201",
				Body:   "hello world\n",
				Header: map[string]string{
					"X-Test": "coucou",
				},
			},
			Expected: Response{
				StatusCode: http.StatusCreated,
				Body: strings.Join([]string{
					"<h1>Requested URL:</h1>",
					"<p>/test</p>",
					"<h1>Request Method:</h1>",
					"<p>POST</p>",
					"<h1>Headers:</h1>",
					"<pre>",
					"Content-Length: 12",
					"Content-Type: text/plain; charset=utf-8",
					"X-Test: coucou",
					"</pre>",
					"<h1>Body:</h1>",
					"<pre>",
					"hello world",
					"</pre>",
				}, "\n"),
				Header: map[string]string{
					"Content-type":  "text/html; charset=UTF-8",
					"X-Powered-By":  "PHP/8.3.7",
					"X-Request-Uri": "/test",
					"Status":        "201 Created",
					"X-Status-Code": "201",
					"Status-Code":   "201",
				},
			},
			Error: "",
		},
	}

	h := fcgiHandler(Server{
		DocumentRoot: dir,
		Index:        "index.php",
		FCGIHost:     "127.0.0.1:9000",
	})

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest(tt.In.Method, tt.In.URL, strings.NewReader(tt.In.Body))
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}
			for key, value := range tt.In.Header {
				req.Header.Add(key, value)
			}

			rr := httptest.NewRecorder()
			stderr, err := h(rr, req)
			if err != nil {
				t.Fatalf("failed send fcgi request: %v", err)
			}
			if err != nil {
				t.Fatalf("fcgi write on stderr: %v", string(stderr))
			}

			if status := rr.Code; status != tt.Expected.StatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.Expected.StatusCode)
			}

			responseBody, err := io.ReadAll(rr.Body)
			if err != nil {
				t.Fatalf("could not read response body: %v", err)
			}
			if strings.TrimSpace(string(responseBody)) != tt.Expected.Body {
				t.Fatalf("handler returned unexpected body: got %v want %v", string(responseBody), tt.Expected.Body)
			}

			for key, expectedValue := range tt.Expected.Header {
				if rr.Header().Get(key) != expectedValue {
					t.Errorf("handler returned wrong header for %s: got %v want %v", key, rr.Header().Get(key), expectedValue)
				}
			}
		})
	}
}
