package decoder

import (
	"app/fcgiclient"
	"bytes"
	"encoding/base64"
	"reflect"
	"strings"
	"testing"
)

// Assume writePairs and readPairs are in the same package or imported accordingly
func TestDecodeEnv(t *testing.T) {
	tests := map[string]struct {
		Data map[string]string
	}{
		"simple case": {
			Data: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		"empty case": {
			Data: map[string]string{},
		},
		"complex case": {
			Data: map[string]string{
				"key1":           "value1",
				"longkey2":       "longvalue2",
				"key with space": "value with space",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer

			// Encode the map
			err := fcgiclient.BuildPair(&buf, tt.Data)
			if err != nil {
				t.Fatalf("writePairs failed: %v", err)
			}

			// Decode the map
			decodedPairs, err := decodeEnv(&buf)
			if err != nil {
				t.Fatalf("readPairs failed: %v", err)
			}

			// Check if the original map and the decoded map are equal
			if !reflect.DeepEqual(tt.Data, decodedPairs) {
				t.Fatalf("decoded map does not match the original map: got %v, want %v", decodedPairs, tt.Data)
			}
		})
	}
}

func TestParseResponse(t *testing.T) {
	tests := map[string]struct {
		In  []string
		Out Response
	}{
		"simple case": {
			In: []string{
				"U3RhdHVzOiA0MTIgUHJlY29uZGl0aW9uIEZhaWxlZA0KU3RhdHVzLUNvZGU6NDEyDQ",
				"pYLVN0YXR1cy1Db2RlOiA0MTINCkNvbnRlbnQtdHlwZTogdGV4dC9odG1sOyBjaGFy",
				"c2V0PVVURi04DQoNCjxoMT5SZXF1ZXN0ZWQgVVJMOjwvaDE+PHA+L3Rlc3Q8L3A+PG",
				"gxPlJlcXVlc3QgTWV0aG9kOjwvaDE+PHA+R0VUPC9wPjxoMT5IZWFkZXJzOjwvaDE+",
				"PHByZT5BY2NlcHQ6ICovKgpVc2VyLUFnZW50OiBjdXJsLzguNi4wCkhvc3Q6IDEyNy",
				"4wLjAuMTo0NTE5CkNvbnRlbnQtVHlwZTogCkNvbnRlbnQtTGVuZ3RoOiAKPC9wcmU+",
				"PGgxPkJvZHk6PC9oMT48cHJlPjwvcHJlPg==",
			},
			Out: Response{
				StatusCode: 412,
				Headers: map[string]string{
					"Status":        "412 Precondition Failed",
					"Status-Code":   "412",
					"X-Status-Code": "412",
					"Content-type":  "text/html; charset=UTF-8",
				},
				Stdout: strings.Join([]string{
					"<h1>Requested URL:</h1><p>/test</p><h1>Request Method:</h1><p>GET</p><h1>Headers:</h1><pre>Accept: */*",
					"User-Agent: curl/8.6.0",
					"Host: 127.0.0.1:4519",
					"Content-Type: ",
					"Content-Length: ",
					"</pre><h1>Body:</h1><pre></pre>",
				}, "\n"),
			},
		},
		"cors": {
			In: []string{
				"WC1Qb3dlcmVkLUJ5OiBQSFAvOC4zLjgNCkNhY2hlLUNvbnRyb2w6IG1heC1hZ2U9MCwg",
				"bXVzdC1yZXZhbGlkYXRlLCBwcml2YXRlDQpEYXRlOiBXZWQsIDE5IEp1biAyMDI0IDA3",
				"OjIxOjExIEdNVA0KVmFyeTogT3JpZ2luDQpBY2Nlc3MtQ29udHJvbC1BbGxvdy1DcmVk",
				"ZW50aWFsczogdHJ1ZQ0KQWNjZXNzLUNvbnRyb2wtQWxsb3ctTWV0aG9kczogUE9TVCwg",
				"UFVULCBQQVRDSCwgR0VULCBERUxFVEUNCkFjY2Vzcy1Db250cm9sLUFsbG93LUhlYWRl",
				"cnM6IGNvbnRlbnQtdHlwZQ0KQWNjZXNzLUNvbnRyb2wtTWF4LUFnZTogMzYwMA0KQWNj",
				"ZXNzLUNvbnRyb2wtQWxsb3ctT3JpZ2luOiBodHRwczovL3ZlcmlmaWNhdGlvbi5leGVt",
				"cGxlLmNvbQ0KQ29udGVudC1TZWN1cml0eS1Qb2xpY3k6IGRlZmF1bHQtc3JjICdub25l",
				"JzsgZnJhbWUtYW5jZXN0b3JzICdub25lJw0KU3RyaWN0LVRyYW5zcG9ydC1TZWN1cml0",
				"eTogIG1heC1hZ2U9NjMwNzIwMDANClgtQ29udGVudC1UeXBlLU9wdGlvbnM6IG5vc25p",
				"ZmYNCkNvbnRlbnQtVHlwZTogdGV4dC9odG1sOyBjaGFyc2V0PVVURi04DQpFeHBpcmVz",
				"OiBXZWQsIDE5IEp1biAyMDI0IDA3OjIxOjExIEdNVA0KDQoAAAAA",
			},
			Out: Response{
				StatusCode: 200,
				Headers: map[string]string{
					"X-Powered-By":                     "PHP/8.3.8",
					"Cache-Control":                    "max-age=0, must-revalidate, private",
					"Date":                             "Wed, 19 Jun 2024 07:21:11 GMT",
					"Vary":                             "Origin",
					"Access-Control-Allow-Credentials": "true",
					"Access-Control-Allow-Methods":     "POST, PUT, PATCH, GET, DELETE",
					"Access-Control-Allow-Headers":     "content-type",
					"Access-Control-Max-Age":           "3600",
					"Access-Control-Allow-Origin":      "https://verification.exemple.com",
					"Content-Security-Policy":          "default-src 'none'; frame-ancestors 'none'",
					"Strict-Transport-Security":        "max-age=63072000",
					"X-Content-Type-Options":           "nosniff",
					"Content-Type":                     "text/html; charset=UTF-8",
					"Expires":                          "Wed, 19 Jun 2024 07:21:11 GMT",
				},
				Stdout: "\x00\x00\x00\x00",
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b64 := strings.Join(tt.In, "")
			content, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				t.Fatalf("cannot read input base64 data : %v", err)
			}
			result, err := ParseResponse(string(content))
			if err != nil {
				t.Fatalf("failed parsing content : %v", err)
			}
			if !reflect.DeepEqual(result, tt.Out) {
				t.Fatalf("\ngot  %#v \nwant %#v\n", result, tt.Out)
			}
		})
	}
}
