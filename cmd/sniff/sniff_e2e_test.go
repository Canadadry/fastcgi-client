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
			Logger: strings.Join([]string{
				"Proxy listening on 127.0.0.1:9001, forwarding to 127.0.0.1:9000",
				"handling new TCP client",
				"connected to server",
				"request read raw " + "[{\"Header\":{\"Version\":1,\"Type\":1,\"Id\":1,\"ContentLength\":8,\"PaddingLength\":0,\"Reserved\":0},\"Buf\":\"AAEAAAAAAAA=\"},{\"Header\":{\"Version\":1,\"Type\":4,\"Id\":1,\"ContentLength\":470,\"PaddingLength\":2,\"Reserved\":0},\"Buf\":\"DgJDT05URU5UX0xFTkdUSDM4DBBDT05URU5UX1RZUEVhcHBsaWNhdGlvbi9qc29uDTRET0NVTUVOVF9ST09UL1VzZXJzL2plcm9tZS9Qcm9nL0VWQ0svdG9vbHMvZmFzdGNnaS1jbGllbnQvcGhwLWZwbQwQRE9DVU1FTlRfVVJJL2FwaS9hdXRoLXRva2VucxEHR0FURVdBWV9JTlRFUkZBQ0VDR0kvMS4xERBIVFRQX0NPTlRFTlRfVFlQRWFwcGxpY2F0aW9uL2pzb24MD1FVRVJZX1NUUklOR3N0YXR1c19jb2RlPTIwMQ4EUkVRVUVTVF9NRVRIT0RQT1NUDgRSRVFVRVNUX1NDSEVNRWh0dHALEFJFUVVFU1RfVVJJL2FwaS9hdXRoLXRva2Vucw8+U0NSSVBUX0ZJTEVOQU1FL1VzZXJzL2plcm9tZS9Qcm9nL0VWQ0svdG9vbHMvZmFzdGNnaS1jbGllbnQvcGhwLWZwbS9pbmRleC5waHALEFNDUklQVF9OQU1FL2FwaS9hdXRoLXRva2Vucw8IU0VSVkVSX1BST1RPQ09MSFRUUC8xLjEPEFNFUlZFUl9TT0ZUV0FSRWdvIC8gZmNnaWNsaWVudCAAAA==\"},{\"Header\":{\"Version\":1,\"Type\":4,\"Id\":1,\"ContentLength\":0,\"PaddingLength\":0,\"Reserved\":0},\"Buf\":\"\"},{\"Header\":{\"Version\":1,\"Type\":5,\"Id\":1,\"ContentLength\":38,\"PaddingLength\":2,\"Reserved\":0},\"Buf\":\"eyJsb2dpbiI6YWRtaW4iLCJwYXNzd29yZCI6ImF6ZXJ0eXUifQoAAA==\"}]",
				"decoded request " + "{\"ReqId\":1,\"Env\":{\"CONTENT_LENGTH\":\"38\",\"CONTENT_TYPE\":\"application/json\",\"DOCUMENT_ROOT\":\"/Users/jerome/Prog/EVCK/tools/fastcgi-client/php-fpm\",\"DOCUMENT_URI\":\"/api/auth-tokens\",\"GATEWAY_INTERFACE\":\"CGI/1.1\",\"HTTP_CONTENT_TYPE\":\"application/json\",\"QUERY_STRING\":\"status_code=201\",\"REQUEST_METHOD\":\"POST\",\"REQUEST_SCHEME\":\"http\",\"REQUEST_URI\":\"/api/auth-tokens\",\"SCRIPT_FILENAME\":\"/Users/jerome/Prog/EVCK/tools/fastcgi-client/php-fpm/index.php\",\"SCRIPT_NAME\":\"/api/auth-tokens\",\"SERVER_PROTOCOL\":\"HTTP/1.1\",\"SERVER_SOFTWARE\":\"go / fcgiclient \"},\"Stdin\":\"eyJsb2dpbiI6YWRtaW4iLCJwYXNzd29yZCI6ImF6ZXJ0eXUifQo=\"}\nwriting back request\nfinish writing to server, waiting for response\nresponse read raw [{\"Header\":{\"Version\":1,\"Type\":6,\"Id\":1,\"ContentLength\":389,\"PaddingLength\":3,\"Reserved\":0},\"Buf\":\"U3RhdHVzOiAyMDEgQ3JlYXRlZA0KWC1Qb3dlcmVkLUJ5OiBQSFAvOC4zLjcNClN0YXR1cy1Db2RlOjIwMQ0KWC1TdGF0dXMtQ29kZTogMjAxDQpYLVJlcXVlc3QtVXJpOiAvYXBpL2F1dGgtdG9rZW5zDQpDb250ZW50LXR5cGU6IHRleHQvaHRtbDsgY2hhcnNldD1VVEYtOA0KDQo8aDE+UmVxdWVzdGVkIFVSTDo8L2gxPgo8cD4vYXBpL2F1dGgtdG9rZW5zPC9wPgo8aDE+UmVxdWVzdCBNZXRob2Q6PC9oMT4KPHA+UE9TVDwvcD4KPGgxPkhlYWRlcnM6PC9oMT4KPHByZT4KQ29udGVudC1MZW5ndGg6IDM4CkNvbnRlbnQtVHlwZTogYXBwbGljYXRpb24vanNvbgo8L3ByZT4KPGgxPkJvZHk6PC9oMT4KPHByZT4KeyJsb2dpbiI6YWRtaW4iLCJwYXNzd29yZCI6ImF6ZXJ0eXUifQo8L3ByZT4AAAA=\"},{\"Header\":{\"Version\":1,\"Type\":3,\"Id\":1,\"ContentLength\":8,\"PaddingLength\":0,\"Reserved\":0},\"Buf\":\"AAAAAABhdGk=\"}]",
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
			go buildServerAndRun(done, printf, "127.0.0.1:9001", "127.0.0.1:9000")
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
