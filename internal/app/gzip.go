package app

import (
	"compress/gzip"
	"io"
	"net/http"
	"slices"
	"strings"
)

type (
	gzipWriter struct {
		w   http.ResponseWriter
		gzw *gzip.Writer
	}

	gzipReader struct {
		r   io.Reader
		gzr *gzip.Reader
	}
)

func (gw gzipWriter) Header() http.Header {
	return gw.w.Header()
}

func (gw gzipWriter) Write(b []byte) (int, error) {
	return gw.gzw.Write(b)
}

func (gw gzipWriter) WriteHeader(statusCode int) {
	gw.Header().Set("Content-Encoding", "gzip")
	gw.w.WriteHeader(statusCode)
}

func (gr gzipReader) Close() error {
	return gr.gzr.Close()
}

func (gr gzipReader) Read(p []byte) (n int, err error) {
	return gr.gzr.Read(p)
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origRW := w

		gzipAllowed := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		contentTypes := []string{"application/json", "text/html"}
		cTypeAllowed := slices.IndexFunc(contentTypes, func(cType string) bool {
			return strings.Contains(r.Header.Get("Content-Type"), cType)
		})

		if gzipAllowed && cTypeAllowed != -1 {
			gzw := gzip.NewWriter(w)
			origRW = gzipWriter{w, gzw}
			defer gzw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		if strings.Contains(contentEncoding, "gzip") {
			gzr, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = gzipReader{r.Body, gzr}
			defer r.Body.Close()
		}

		next.ServeHTTP(origRW, r)
	})
}
