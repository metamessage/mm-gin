package mmvanilla

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"reflect"

	mm "github.com/metamessage/metamessage"
)

const (
	ContentTypeMetaMessage = "application/x-metamessage"
	ContentTypeJSONC       = "application/jsonc"
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

func Init(mux *http.ServeMux, prefix string) {
	defaultMux = mux
	defaultPrefix = prefix
}

func GET(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	if len(handlers) == 1 {
		h := handlers[0]
		defaultMux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				wrappedAbort(w, r, http.StatusMethodNotAllowed, mmError{Error: "method not allowed"})
				return
			}
			h(w, r)
		})
	} else if len(handlers) > 1 {
		defaultMux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				wrappedAbort(w, r, http.StatusMethodNotAllowed, mmError{Error: "method not allowed"})
				return
			}
			h := chain(handlers)
			h(w, r)
		})
	}
}

func HEAD(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := handlers[0]
	defaultMux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			wrappedAbort(w, r, http.StatusMethodNotAllowed, mmError{Error: "method not allowed"})
			return
		}
		h(w, r)
	})
}

func DELETE(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := handlers[0]
	defaultMux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			wrappedAbort(w, r, http.StatusMethodNotAllowed, mmError{Error: "method not allowed"})
			return
		}
		h(w, r)
	})
}

func OPTIONS(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := handlers[0]
	defaultMux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions {
			wrappedAbort(w, r, http.StatusMethodNotAllowed, mmError{Error: "method not allowed"})
			return
		}
		h(w, r)
	})
}

func Any(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := handlers[0]
	defaultMux.HandleFunc(fullPath, h)
}

type Handler[T any] func(r *http.Request, req *T) (data any, err error)

func POST[T any](relativePath string, handler Handler[T]) {
	fullPath := defaultPrefix + relativePath
	defaultMux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			var sample T
			initSafeDefaults(&sample)
			encoded, err := mm.EncodeFromValue(sample, "")
			if err != nil {
				wrappedAbort(w, r, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
				return
			}
			w.Header().Set("Allow", "POST, OPTIONS")
			w.Header().Set("Content-Type", ContentTypeMetaMessage)
			w.WriteHeader(http.StatusOK)
			w.Write(encoded)
			return
		}
		if r.Method != http.MethodPost {
			wrappedAbort(w, r, http.StatusMethodNotAllowed, mmError{Error: "method not allowed"})
			return
		}
		var req T
		if err := bind(r, &req); err != nil {
			wrappedAbort(w, r, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			wrappedAbort(w, r, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		wrappedRespond(w, r, data, "")
	})
}

func PUT[T any](relativePath string, handler Handler[T]) {
	fullPath := defaultPrefix + relativePath
	defaultMux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			var sample T
			initSafeDefaults(&sample)
			encoded, err := mm.EncodeFromValue(sample, "")
			if err != nil {
				wrappedAbort(w, r, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
				return
			}
			w.Header().Set("Allow", "PUT, OPTIONS")
			w.Header().Set("Content-Type", ContentTypeMetaMessage)
			w.WriteHeader(http.StatusOK)
			w.Write(encoded)
			return
		}
		if r.Method != http.MethodPut {
			wrappedAbort(w, r, http.StatusMethodNotAllowed, mmError{Error: "method not allowed"})
			return
		}
		var req T
		if err := bind(r, &req); err != nil {
			wrappedAbort(w, r, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			wrappedAbort(w, r, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		wrappedRespond(w, r, data, "")
	})
}

func PATCH[T any](relativePath string, handler Handler[T]) {
	fullPath := defaultPrefix + relativePath
	defaultMux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			var sample T
			initSafeDefaults(&sample)
			encoded, err := mm.EncodeFromValue(sample, "")
			if err != nil {
				wrappedAbort(w, r, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
				return
			}
			w.Header().Set("Allow", "PATCH, OPTIONS")
			w.Header().Set("Content-Type", ContentTypeMetaMessage)
			w.WriteHeader(http.StatusOK)
			w.Write(encoded)
			return
		}
		if r.Method != http.MethodPatch {
			wrappedAbort(w, r, http.StatusMethodNotAllowed, mmError{Error: "method not allowed"})
			return
		}
		var req T
		if err := bind(r, &req); err != nil {
			wrappedAbort(w, r, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			wrappedAbort(w, r, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		wrappedRespond(w, r, data, "")
	})
}

func bind(r *http.Request, obj any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	contentType := r.Header.Get("Content-Type")
	if contentType == ContentTypeMetaMessage || isBinary(body) {
		return mm.DecodeToValue(body, obj)
	}
	return mm.JsoncToValue(string(body), obj)
}

func isBinary(data []byte) bool {
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

func wrappedRespond(w http.ResponseWriter, r *http.Request, data any, tag string) {
	if mw, ok := w.(*mmResponseWriter); ok {
		mw.respond(data, tag)
	}
}

func wrappedRespondWithStatus(w http.ResponseWriter, r *http.Request, code int, data any, tag string) {
	if mw, ok := w.(*mmResponseWriter); ok {
		mw.respondWithStatus(code, data, tag)
	}
}

func wrappedAbort(w http.ResponseWriter, r *http.Request, code int, obj any) {
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