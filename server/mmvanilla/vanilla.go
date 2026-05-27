package mmvanilla

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/metamessage/web"

	mm "github.com/metamessage/metamessage"
)

var defaultMux *http.ServeMux
var defaultPrefix string

type mmError struct {
	Error string `mm:"desc=Error info"`
}

type mmResponseWriter struct {
	http.ResponseWriter
	data       any
	tag        string
	code       int
	statusCode int
	body       bytes.Buffer
	aborted    bool
}

func (w *mmResponseWriter) WriteHeader(code int) {
	w.statusCode = code
}

func (w *mmResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *mmResponseWriter) respond(data any, tag string) {
	w.data = data
	w.tag = tag
}

func (w *mmResponseWriter) respondWithStatus(code int, data any, tag string) {
	w.data = data
	w.tag = tag
	w.code = code
}

func (w *mmResponseWriter) abortWithMetaMessage(code int, obj any) {
	w.data = obj
	w.code = code
	w.aborted = true
}

func encodeResponse(w http.ResponseWriter, mw *mmResponseWriter) {
	if mw.aborted {
		encoded, err := mm.EncodeFromValue(mw.data, "")
		if err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
			return
		}
		code := mw.code
		if code == 0 {
			code = http.StatusOK
		}
		w.Header().Set("Content-Type", web.ContentTypeMetaMessage)
		w.WriteHeader(code)
		w.Write(encoded)
		return
	}

	if mw.data != nil {
		encoded, err := mm.EncodeFromValue(mw.data, mw.tag)
		if err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
			return
		}
		code := mw.code
		if code == 0 {
			code = http.StatusOK
		}
		w.Header().Set("Content-Type", web.ContentTypeMetaMessage)
		w.WriteHeader(code)
		w.Write(encoded)
		return
	}

	if mw.body.Len() > 0 {
		if mw.statusCode == 0 {
			mw.statusCode = http.StatusOK
		}
		w.WriteHeader(mw.statusCode)
		w.Write(mw.body.Bytes())
	}
}

func wrapHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mw := &mmResponseWriter{ResponseWriter: w}
		h(mw, r)
		encodeResponse(w, mw)
	}
}

func Init(mux *http.ServeMux, prefix string) {
	defaultMux = mux
	defaultPrefix = prefix
}

func GET(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := wrapHandler(handlers[0])
	defaultMux.HandleFunc("GET "+fullPath, h)
}

func HEAD(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := wrapHandler(handlers[0])
	defaultMux.HandleFunc("HEAD "+fullPath, h)
}

func DELETE(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := wrapHandler(handlers[0])
	defaultMux.HandleFunc("DELETE "+fullPath, h)
}

func OPTIONS(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := wrapHandler(handlers[0])
	defaultMux.HandleFunc("OPTIONS "+fullPath, h)
}

func Any(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := wrapHandler(handlers[0])
	defaultMux.HandleFunc(fullPath, h)
}

type Handler[T any] func(r *http.Request, req *T) (data any, err error)

func POST[T any](relativePath string, handler Handler[T]) {
	fullPath := defaultPrefix + relativePath
	defaultMux.HandleFunc("POST "+fullPath, func(w http.ResponseWriter, r *http.Request) {
		mw := &mmResponseWriter{ResponseWriter: w}
		var req T
		if err := bind(r, &req); err != nil {
			mw.abortWithMetaMessage(http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			encodeResponse(w, mw)
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			mw.abortWithMetaMessage(http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			encodeResponse(w, mw)
			return
		}
		mw.respond(data, "")
		encodeResponse(w, mw)
	})
	defaultMux.HandleFunc("OPTIONS "+fullPath, func(w http.ResponseWriter, r *http.Request) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(w, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
			return
		}
		w.Header().Set("Allow", "POST, OPTIONS")
		w.Header().Set("Content-Type", web.ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func PUT[T any](relativePath string, handler Handler[T]) {
	fullPath := defaultPrefix + relativePath
	defaultMux.HandleFunc("PUT "+fullPath, func(w http.ResponseWriter, r *http.Request) {
		mw := &mmResponseWriter{ResponseWriter: w}
		var req T
		if err := bind(r, &req); err != nil {
			mw.abortWithMetaMessage(http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			encodeResponse(w, mw)
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			mw.abortWithMetaMessage(http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			encodeResponse(w, mw)
			return
		}
		mw.respond(data, "")
		encodeResponse(w, mw)
	})

	defaultMux.HandleFunc("OPTIONS "+fullPath, func(w http.ResponseWriter, r *http.Request) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(w, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
			return
		}
		w.Header().Set("Allow", "PUT, OPTIONS")
		w.Header().Set("Content-Type", web.ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func PATCH[T any](relativePath string, handler Handler[T]) {
	fullPath := defaultPrefix + relativePath
	defaultMux.HandleFunc("PATCH "+fullPath, func(w http.ResponseWriter, r *http.Request) {
		mw := &mmResponseWriter{ResponseWriter: w}
		var req T
		if err := bind(r, &req); err != nil {
			mw.abortWithMetaMessage(http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			encodeResponse(w, mw)
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			mw.abortWithMetaMessage(http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			encodeResponse(w, mw)
			return
		}
		mw.respond(data, "")
		encodeResponse(w, mw)
	})
	defaultMux.HandleFunc("OPTIONS "+fullPath, func(w http.ResponseWriter, r *http.Request) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(w, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
			return
		}
		w.Header().Set("Allow", "PATCH, OPTIONS")
		w.Header().Set("Content-Type", web.ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func bind(r *http.Request, obj any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	if isBinaryMetaMessage(body) {
		return mm.DecodeToValue(body, obj)
	}
	return mm.JsoncToValue(string(body), obj)
}

func isBinaryMetaMessage(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	c := data[0]
	return c != '{' && c != '[' && c != '"'
}

func chain(handlers []http.HandlerFunc) http.HandlerFunc {
	if len(handlers) == 0 {
		return func(w http.ResponseWriter, r *http.Request) {}
	}
	if len(handlers) == 1 {
		return handlers[0]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var index int
		var next func()
		next = func() {
			if index < len(handlers) {
				h := handlers[index]
				index++
				h(w, r)
			}
		}
		next()
	}
}

func Respond(w http.ResponseWriter, data any, tag string) {
	if mw, ok := w.(*mmResponseWriter); ok {
		mw.respond(data, tag)
	}
}

func RespondWithStatus(w http.ResponseWriter, code int, data any, tag string) {
	if mw, ok := w.(*mmResponseWriter); ok {
		mw.respondWithStatus(code, data, tag)
	}
}

func AbortWithMetaMessage(w http.ResponseWriter, code int, obj any) {
	if mw, ok := w.(*mmResponseWriter); ok {
		mw.abortWithMetaMessage(code, obj)
		return
	}
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", web.ContentTypeMetaMessage)
	w.WriteHeader(code)
	w.Write(encoded)
}
