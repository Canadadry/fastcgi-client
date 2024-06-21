package middleware

import "net/http"

type wrapWriter struct {
	w           http.ResponseWriter
	statusCode  int
	byteWritten int
}

func (ww *wrapWriter) Header() http.Header {
	return ww.w.Header()
}
func (ww *wrapWriter) Write(b []byte) (int, error) {
	n, err := ww.w.Write(b)
	ww.byteWritten += n
	return n, err
}
func (ww *wrapWriter) WriteHeader(statusCode int) {
	ww.statusCode = statusCode
	ww.w.WriteHeader(statusCode)
}
