package mmvanilla

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/metamessage/mm-web-go/web"

	mm "github.com/metamessage/metamessage"
)

var defaultMux *http.ServeMux
var defaultPrefix string

type mmError struct {
	Error string `mm:"desc=Error info"`
}

type mmResponseWriter struct {
	http.ResponseWriter
	data        any
	tag         string
	code        int
	statusCode  int
	body        bytes.Buffer
	aborted     bool
	wroteHeader bool
}

func (w *mmResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
	}
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

func HEAD(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := wrapHandler(chain(handlers))
	defaultMux.HandleFunc("HEAD "+fullPath, h)
}

func OPTIONS(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := wrapHandler(chain(handlers))
	defaultMux.HandleFunc("OPTIONS "+fullPath, h)
}

func Any(relativePath string, handlers ...http.HandlerFunc) {
	fullPath := defaultPrefix + relativePath
	h := wrapHandler(chain(handlers))
	defaultMux.HandleFunc(fullPath, h)
}

type Handler[T any] func(r *http.Request, req *T) (data any, err error)

type HandlerWithoutReq func(r *http.Request) (data any, err error)

func registerHandler[T any](method, relativePath string, handler Handler[T]) {
	if defaultMux == nil {
		return
	}
	fullPath := defaultPrefix + relativePath
	defaultMux.HandleFunc(fmt.Sprintf("%s %s", method, fullPath), func(w http.ResponseWriter, r *http.Request) {
		mw := &mmResponseWriter{ResponseWriter: w}
		var req T
		if slices.Contains([]string{http.MethodPost, http.MethodPut, http.MethodPatch}, r.Method) {
			if err := bind(r, &req); err != nil {
				mw.abortWithMetaMessage(http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
				encodeResponse(w, mw)
				return
			}
		}

		if slices.Contains([]string{http.MethodGet, http.MethodDelete}, r.Method) {
			if err := bindQuery(r, &req); err != nil {
				mw.abortWithMetaMessage(http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
				encodeResponse(w, mw)
				return
			}
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
	defaultMux.HandleFunc(fmt.Sprintf("%s %s", http.MethodOptions, fullPath), func(w http.ResponseWriter, r *http.Request) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			AbortWithMetaMessage(w, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
			return
		}
		w.Header().Set("Allow", fmt.Sprintf("%s, %s", method, http.MethodOptions))
		w.Header().Set("Content-Type", web.ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func GET[T any](relativePath string, handler Handler[T]) {
	registerHandler(http.MethodGet, relativePath, handler)
}

func DELETE[T any](relativePath string, handler Handler[T]) {
	registerHandler(http.MethodDelete, relativePath, handler)
}

func POST[T any](relativePath string, handler Handler[T]) {
	registerHandler(http.MethodPost, relativePath, handler)
}

func PUT[T any](relativePath string, handler Handler[T]) {
	registerHandler(http.MethodPut, relativePath, handler)
}

func PATCH[T any](relativePath string, handler Handler[T]) {
	registerHandler(http.MethodPatch, relativePath, handler)
}

func bind(r *http.Request, obj any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	ct := r.Header.Get("Content-Type")
	switch ct {
	case web.ContentTypeMetaMessage:
		return mm.DecodeToValue(body, obj)
	case web.ContentTypeJSONC:
		return mm.JsoncToValue(string(body), obj)
	default:
		return fmt.Errorf("unsupported Content-Type: %s", ct)
	}
}

// func bindQuery(r *http.Request, obj any) error {
// 	query := r.URL.RawQuery
// 	if query == "" {
// 		query = "null"
// 	}

// 	return mm.JsoncToValue(query, obj)
// }

func bindQuery(r *http.Request, obj any) error {
	dataHex := r.URL.Query().Get("data")
	if dataHex == "" {
		return nil
	} else {
		binData, err := hex.DecodeString(dataHex)
		if err != nil {
			return fmt.Errorf("invalid hex string")
		}

		return mm.DecodeToValue(binData, obj)
	}
}

func chain(handlers []http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, h := range handlers {
			h(w, r)
		}
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
