package mmchi

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/metamessage/mm-web-go/web"

	chi "github.com/go-chi/chi/v5"
	mm "github.com/metamessage/metamessage"
)

var defaultGroup chi.Router

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

func decoderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead ||
			r.Method == http.MethodDelete || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			AbortWithMetaMessage(w, http.StatusBadRequest, mmError{Error: "read body failed"})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}

func encoderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mw := &mmResponseWriter{ResponseWriter: w}
		next.ServeHTTP(mw, r)

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
	})
}

func Init(r chi.Router, relativePath string) chi.Router {
	r.Use(decoderMiddleware)
	r.Use(encoderMiddleware)
	rg := chi.NewRouter()
	r.Mount(relativePath, rg)
	defaultGroup = rg
	return rg
}

func bindQuery(r *http.Request, obj any) error {
	dataHex := r.URL.Query().Get("data")
	if dataHex == "" {
		return nil
	}
	binData, err := hex.DecodeString(dataHex)
	if err != nil {
		return fmt.Errorf("invalid hex string")
	}
	return mm.DecodeToValue(binData, obj)
}

func GET[T any](relativePath string, handler Handler[T]) {
	mux := defaultGroup.(chi.Router)
	mux.Get(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := bindQuery(r, &req); err != nil {
			AbortWithMetaMessage(w, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, tag, err := handler(r, &req)
		if err != nil {
			AbortWithMetaMessage(w, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		Respond(w, data, tag)
	})
	mux.Options(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(w, http.StatusInternalServerError, mmError{Error: fmt.Sprintf("schema encode failed: %s", err.Error())})
			return
		}
		w.Header().Set("Allow", "GET, OPTIONS")
		w.Header().Set("Content-Type", web.ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func HEAD(relativePath string, handlers ...http.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	defaultGroup.Head(relativePath, handlers[0])
}

func DELETE[T any](relativePath string, handler Handler[T]) {
	mux := defaultGroup.(chi.Router)
	mux.Delete(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := bindQuery(r, &req); err != nil {
			AbortWithMetaMessage(w, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, tag, err := handler(r, &req)
		if err != nil {
			AbortWithMetaMessage(w, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		Respond(w, data, tag)
	})
	mux.Options(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(w, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
			return
		}
		w.Header().Set("Allow", "DELETE, OPTIONS")
		w.Header().Set("Content-Type", web.ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func OPTIONS(relativePath string, handlers ...http.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	defaultGroup.Options(relativePath, handlers[0])
}

func Any(relativePath string, handlers ...http.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	defaultGroup.Handle(relativePath, handlers[0])
}

type Handler[T any] func(r *http.Request, req *T) (data any, tag string, err error)

func POST[T any](relativePath string, handler Handler[T]) {
	mux := defaultGroup.(chi.Router)
	mux.Post(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := bind(r, &req); err != nil {
			AbortWithMetaMessage(w, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, tag, err := handler(r, &req)
		if err != nil {
			AbortWithMetaMessage(w, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		Respond(w, data, tag)
	})
	mux.Options(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(w, http.StatusInternalServerError, mmError{Error: fmt.Sprintf("schema encode failed: %s", err.Error())})
			return
		}
		w.Header().Set("Allow", "POST, OPTIONS")
		w.Header().Set("Content-Type", web.ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func PUT[T any](relativePath string, handler Handler[T]) {
	mux := defaultGroup.(chi.Router)
	mux.Put(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := bind(r, &req); err != nil {
			AbortWithMetaMessage(w, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, tag, err := handler(r, &req)
		if err != nil {
			AbortWithMetaMessage(w, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		Respond(w, data, tag)
	})
	mux.Options(relativePath, func(w http.ResponseWriter, r *http.Request) {
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
	mux := defaultGroup.(chi.Router)
	mux.Patch(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := bind(r, &req); err != nil {
			AbortWithMetaMessage(w, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, tag, err := handler(r, &req)
		if err != nil {
			AbortWithMetaMessage(w, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		Respond(w, data, tag)
	})
	mux.Options(relativePath, func(w http.ResponseWriter, r *http.Request) {
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
