package handler

import (
	"app/pkg/http/middleware"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
)

func Static(folder, index string) func(w http.ResponseWriter, r *http.Request) bool {
	h := http.NewServeMux()
	h.Handle("/", http.FileServer(http.Dir(folder)))

	return func(w http.ResponseWriter, r *http.Request) bool {
		filename := path.Join(folder, r.URL.Path)
		if r.URL.Path == "/.env" || r.URL.Path == "/" || r.URL.Path == "" {
			filename = path.Join(folder, index)
		}
		_, err := os.Stat(filename)
		if err == nil && !strings.HasSuffix(filename, ".php") {
			h.ServeHTTP(w, r)
			return true
		}
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("cannot stat %s : %v\n", filename, err)
			middleware.Respond(w, "server error", 500, nil)
			return true
		}
		return false
	}
}
