package mmchi

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/go-chi/chi/v5"
	mm "github.com/metamessage/metamessage"
)

const (
	ContentTypeMetaMessage = "application/x-metamessage"
	ContentTypeJSONC       = "application/jsonc"
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
			abortWithMetaMessage(w, r, http.StatusBadRequest, mmError{Error: "read body failed"})
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
			w.Header().Set("Content-Type", ContentTypeMetaMessage)
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
			w.Header().Set("Content-Type", ContentTypeMetaMessage)
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

func GET(relativePath string, handlers ...http.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	defaultGroup.Get(relativePath, handlers[0])
}

func HEAD(relativePath string, handlers ...http.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	defaultGroup.Head(relativePath, handlers[0])
}

func DELETE(relativePath string, handlers ...http.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	defaultGroup.Delete(relativePath, handlers[0])
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

type Handler[T any] func(r *http.Request, req *T) (data any, err error)

func POST[T any](relativePath string, handler Handler[T]) {
	mux := defaultGroup.(chi.Router)
	mux.Post(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := bind(r, &req); err != nil {
			abortWithMetaMessage(w, r, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			abortWithMetaMessage(w, r, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		respond(w, r, data, "")
	})
	mux.Options(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var sample T
		initSafeDefaults(&sample)
		encoded, err := mm.EncodeFromValue(sample, "")
		if err != nil {
			abortWithMetaMessage(w, r, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
			return
		}
		w.Header().Set("Allow", "POST, OPTIONS")
		w.Header().Set("Content-Type", ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func PUT[T any](relativePath string, handler Handler[T]) {
	mux := defaultGroup.(chi.Router)
	mux.Put(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := bind(r, &req); err != nil {
			abortWithMetaMessage(w, r, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			abortWithMetaMessage(w, r, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		respond(w, r, data, "")
	})
	mux.Options(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var sample T
		initSafeDefaults(&sample)
		encoded, err := mm.EncodeFromValue(sample, "")
		if err != nil {
			abortWithMetaMessage(w, r, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
			return
		}
		w.Header().Set("Allow", "PUT, OPTIONS")
		w.Header().Set("Content-Type", ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func PATCH[T any](relativePath string, handler Handler[T]) {
	mux := defaultGroup.(chi.Router)
	mux.Patch(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := bind(r, &req); err != nil {
			abortWithMetaMessage(w, r, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			abortWithMetaMessage(w, r, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		respond(w, r, data, "")
	})
	mux.Options(relativePath, func(w http.ResponseWriter, r *http.Request) {
		var sample T
		initSafeDefaults(&sample)
		encoded, err := mm.EncodeFromValue(sample, "")
		if err != nil {
			abortWithMetaMessage(w, r, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
			return
		}
		w.Header().Set("Allow", "PATCH, OPTIONS")
		w.Header().Set("Content-Type", ContentTypeMetaMessage)
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

func respond(w http.ResponseWriter, r *http.Request, data any, tag string) {
	if mw, ok := w.(*mmResponseWriter); ok {
		mw.respond(data, tag)
	}
}

func respondWithStatus(w http.ResponseWriter, r *http.Request, code int, data any, tag string) {
	if mw, ok := w.(*mmResponseWriter); ok {
		mw.respondWithStatus(code, data, tag)
	}
}

func abortWithMetaMessage(w http.ResponseWriter, r *http.Request, code int, obj any) {
	if mw, ok := w.(*mmResponseWriter); ok {
		mw.abortWithMetaMessage(code, obj)
		return
	}
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", ContentTypeMetaMessage)
	w.WriteHeader(code)
	w.Write(encoded)
}

func initSafeDefaults(obj any) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}
		tag := t.Field(i).Tag.Get("mm")
		switch field.Kind() {
		case reflect.String:
			if field.String() == "" {
				field.SetString("x")
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() == 0 {
				field.SetInt(1)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if field.Uint() == 0 {
				field.SetUint(1)
			}
		case reflect.Float32, reflect.Float64:
			if field.Float() == 0 {
				_ = tag
				field.SetFloat(1)
			}
		}
	}
}