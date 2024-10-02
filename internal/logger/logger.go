package logger

import (
	"go.uber.org/zap"
	"net/http"
	"time"
)

var Log *zap.SugaredLogger

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		*responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.status = statusCode
}

func Initialize() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	defer logger.Sync()

	Log = logger.Sugar()
	return nil
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &loggingResponseWriter{w, &responseData{}}

		next.ServeHTTP(lrw, r)

		Log.Infow("user request",
			"uri", r.RequestURI,
			"method", r.Method,
			"duration", time.Since(start),
			"status", lrw.status,
			"size", lrw.size,
		)
	})
}
