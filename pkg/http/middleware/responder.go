package middleware

import (
	"fmt"
	"net/http"
)

func Respond(w http.ResponseWriter, body string, statusCode int, headers map[string]string) {
	for header, value := range headers {
		w.Header().Set(header, value)
	}
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, "%s", body)
}
