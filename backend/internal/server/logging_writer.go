package server

import (
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
)

type loggingResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func newLoggingResponseWriter(w gin.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{ResponseWriter: w, body: &bytes.Buffer{}}
}

func (w *loggingResponseWriter) Write(p []byte) (int, error) {
	w.body.Write(p)
	return w.ResponseWriter.Write(p)
}

func (w *loggingResponseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) Body() string {
	return w.body.String()
}

func (w *loggingResponseWriter) ErrorMessage() string {
	body := w.body.String()
	if body == "" {
		return ""
	}
	var generic map[string]any
	if err := json.Unmarshal([]byte(body), &generic); err != nil {
		return body
	}
	if msg, ok := generic["error"].(string); ok {
		return msg
	}
	if errObj, ok := generic["error"].(map[string]any); ok {
		if msg, ok := errObj["message"].(string); ok {
			return msg
		}
	}
	return body
}
