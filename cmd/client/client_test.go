package client

import (
	"net/url"
	"os"
	"os/exec"
	"path"
	"reflect"
	"testing"
	"time"
)

func runPhpFpmServer(t *testing.T) (string, func() error) {
	t.Helper()
	cwd, _ := os.Getwd()
	t.Log(cwd)
	cwd = path.Join(cwd, "../../php-fpm")
	t.Log(cwd)
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
		In       FCGIRequest
		Expected FCGIResponse
	}{
		"main stript not found": {
			In: FCGIRequest{
				DocumentRoot: dir,
				Method:       "GET",
				Url:          MustUrl(t, "/"),
				Body:         "",
				Index:        "wrong.php",
				Env:          map[string]string{},
				Header:       map[string]string{},
			},
			Expected: FCGIResponse{
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
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := Do("127.0.0.1:9000", tt.In)
			if err != nil {
				t.Fatalf("failed running request : %v", err)
			}
			if !reflect.DeepEqual(tt.Expected, result) {
				t.Fatalf("want \n%#v\ngot \n%#v\n", tt.Expected, result)
			}
		})
	}

}
