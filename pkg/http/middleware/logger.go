package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func HandleWithLogAndError(next func(w http.ResponseWriter, r *http.Request) ([]byte, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		ww := &wrapWriter{w: rw, statusCode: http.StatusOK}
		startedAt := time.Now()
		stderr, err := next(ww, r)
		endedAt := time.Now()
		logLine := struct {
			At           time.Time
			Level        string
			Method       string
			URL          string
			Status       int
			Bytes        int
			Elapsed      string
			ErrorMessage string
			Stderr       []byte
		}{
			At:           startedAt,
			Level:        "trace",
			Method:       r.Method,
			URL:          r.URL.String(),
			Status:       ww.statusCode,
			Bytes:        ww.byteWritten,
			Elapsed:      fmt.Sprintf("%v", endedAt.Sub(startedAt)),
			ErrorMessage: "",
			Stderr:       stderr,
		}
		if len(stderr) > 0 {
			logLine.Level = "error"
		}
		if logLine.Status >= 500 {
			logLine.Level = "error"
		}
		if err != nil {
			logLine.Level = "error"
			logLine.ErrorMessage = err.Error()
			Respond(rw, "server error", 500, nil)
		}
		_ = json.NewEncoder(os.Stdout).Encode(logLine)
	}
}
