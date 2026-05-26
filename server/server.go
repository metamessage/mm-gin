package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"

	"github.com/gin-gonic/gin"
	mm "github.com/metamessage/metamessage"

	mmgin "github.com/metamessage/mm-web"
	"github.com/metamessage/mm-web/server/mmchi"
	"github.com/metamessage/mm-web/server/mmecho"
	"github.com/metamessage/mm-web/server/mmfiber"
	"github.com/metamessage/mm-web/server/mmvanilla"
)

// ContentTypeMetaMessage is the Content-Type for MetaMessage binary format.
const ContentTypeMetaMessage = "application/x-metamessage"

// Handler is a framework-agnostic handler for POST/PUT/PATCH.
// It uses *http.Request instead of framework-specific context types,
// making handler code portable across all supported frameworks.
type Handler[T any] func(r *http.Request, req *T) (data any, err error)

// mmError represents an error response in MetaMessage format.
type mmError struct {
	Error string `mm:"desc=Error info"`
}

// Router unified interface that wraps framework-specific routers.
// handlers are framework-specific types:
//
//	*gin.Engine:    handlers are gin.HandlerFunc
//	*echo.Echo:     handlers are echo.HandlerFunc
//	*fiber.App:     handlers are fiber.Handler
//	chi.Router:     handlers are http.HandlerFunc
//	*http.ServeMux: handlers are http.HandlerFunc
type Router interface {
	Group(relativePath string, handlers ...any) Router
	Handle(method, path string, handlers ...any)
	Use(handlers ...any)
}

type frameworkType int

const (
	frameworkUnknown frameworkType = iota
	frameworkGin
	frameworkEcho
	frameworkFiber
	frameworkChi
	frameworkVanilla
)

var (
	currentFramework frameworkType
	currentGroup     Router
)

// Init detects the router type, sets up MetaMessage middleware,
// and configures a route group at the given prefix.
// After Init, use package-level functions GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS/Any
// to register routes on the configured group.
// Supported router types: *gin.Engine, *echo.Echo, *fiber.App, chi.Router, *http.ServeMux
func Init(router any, prefix string) error {
	switch r := router.(type) {
	case *gin.Engine:
		return initGin(r, prefix)
	case *echo.Echo:
		return initEcho(r, prefix)
	case *fiber.App:
		return initFiber(r, prefix)
	case chi.Router:
		return initChi(r, prefix)
	case *http.ServeMux:
		return initVanilla(r, prefix)
	default:
		return fmt.Errorf("unsupported router type: %T", router)
	}
}

// decodeBody decodes MetaMessage binary or JSONC data into the target object.
// It auto-detects the format.
func decodeBody(data []byte, obj any) error {
	if len(data) == 0 {
		return nil
	}
	if isBinaryMetaMessage(data) {
		return mm.DecodeToValue(data, obj)
	}
	return mm.JsoncToValue(string(data), obj)
}

// isBinaryMetaMessage detects whether the data is in MetaMessage binary format.
func isBinaryMetaMessage(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	c := data[0]
	return c != '{' && c != '[' && c != '"'
}

// ---------------------------------------------------------------------------
// Gin adapter
// ---------------------------------------------------------------------------

type ginRouter struct {
	group *gin.RouterGroup
}

func (r *ginRouter) Group(relativePath string, handlers ...any) Router {
	ginHandlers := make([]gin.HandlerFunc, len(handlers))
	for i, h := range handlers {
		ginHandlers[i] = h.(gin.HandlerFunc)
	}
	return &ginRouter{group: r.group.Group(relativePath, ginHandlers...)}
}

func (r *ginRouter) Handle(method, path string, handlers ...any) {
	ginHandlers := make([]gin.HandlerFunc, len(handlers))
	for i, h := range handlers {
		ginHandlers[i] = h.(gin.HandlerFunc)
	}
	r.group.Handle(method, path, ginHandlers...)
}

func (r *ginRouter) Use(handlers ...any) {
	ginHandlers := make([]gin.HandlerFunc, len(handlers))
	for i, h := range handlers {
		ginHandlers[i] = h.(gin.HandlerFunc)
	}
	r.group.Use(ginHandlers...)
}

func initGin(r *gin.Engine, prefix string) error {
	currentFramework = frameworkGin
	rg := mmgin.Init(r, prefix)
	currentGroup = &ginRouter{group: rg}
	return nil
}

func ginDo[T any](path string, method string, handler Handler[T]) {
	rg := currentGroup.(*ginRouter)
	rg.group.Handle(method, path, func(c *gin.Context) {
		var req T
		if err := mmgin.Bind(c, &req); err != nil {
			mmgin.AbortWithMetaMessage(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, err := handler(c.Request, &req)
		if err != nil {
			mmgin.AbortWithMetaMessage(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		encoded, err := mm.EncodeFromValue(data, "")
		if err != nil {
			mmgin.AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "encode failed"})
			return
		}
		c.Data(http.StatusOK, ContentTypeMetaMessage, encoded)
	})
	rg.group.OPTIONS(path, func(c *gin.Context) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			mmgin.AbortWithMetaMessage(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
			return
		}
		c.Header("Allow", method+", OPTIONS")
		c.Data(http.StatusOK, ContentTypeMetaMessage, encoded)
	})
}

// ---------------------------------------------------------------------------
// Echo adapter
// ---------------------------------------------------------------------------

type echoRouter struct {
	group *echo.Group
}

func (r *echoRouter) Group(relativePath string, handlers ...any) Router {
	echoHandlers := make([]echo.MiddlewareFunc, len(handlers))
	for i, h := range handlers {
		echoHandlers[i] = h.(echo.MiddlewareFunc)
	}
	return &echoRouter{group: r.group.Group(relativePath, echoHandlers...)}
}

func (r *echoRouter) Handle(method, path string, handlers ...any) {
	if len(handlers) == 0 {
		return
	}
	h := handlers[0].(echo.HandlerFunc)
	mws := make([]echo.MiddlewareFunc, len(handlers)-1)
	for i, m := range handlers[1:] {
		mws[i] = m.(echo.MiddlewareFunc)
	}
	r.group.Add(method, path, h, mws...)
}

func (r *echoRouter) Use(handlers ...any) {
	echoHandlers := make([]echo.MiddlewareFunc, len(handlers))
	for i, h := range handlers {
		echoHandlers[i] = h.(echo.MiddlewareFunc)
	}
	r.group.Use(echoHandlers...)
}

func initEcho(e *echo.Echo, prefix string) error {
	currentFramework = frameworkEcho
	g := mmecho.Init(e, prefix)
	currentGroup = &echoRouter{group: g}
	return nil
}

func echoDo[T any](path string, method string, handler Handler[T]) {
	rg := currentGroup.(*echoRouter)
	rg.group.Add(method, path, func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echoAbort(c, http.StatusBadRequest, mmError{Error: "read body failed: " + err.Error()})
		}
		c.Request().Body = io.NopCloser(bytes.NewReader(body))

		var req T
		if err := decodeBody(body, &req); err != nil {
			return echoAbort(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		data, err := handler(c.Request(), &req)
		if err != nil {
			return echoAbort(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		encoded, err := mm.EncodeFromValue(data, "")
		if err != nil {
			return echoAbort(c, http.StatusInternalServerError, mmError{Error: "encode failed"})
		}
		return c.Blob(http.StatusOK, ContentTypeMetaMessage, encoded)
	})
	rg.group.OPTIONS(path, func(c echo.Context) error {
		return optionsSchema[T](c, method)
	})
}

func echoAbort(c echo.Context, code int, obj any) error {
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		return c.String(http.StatusInternalServerError, "encode error")
	}
	return c.Blob(code, ContentTypeMetaMessage, encoded)
}

func optionsSchema[T any](c echo.Context, method string) error {
	var sample T
	encoded, err := mm.EncodeFromValue(sample, "example")
	if err != nil {
		return echoAbort(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
	}
	c.Response().Header().Set("Allow", method+", OPTIONS")
	return c.Blob(http.StatusOK, ContentTypeMetaMessage, encoded)
}

// ---------------------------------------------------------------------------
// Fiber adapter
// ---------------------------------------------------------------------------

type fiberRouter struct {
	group fiber.Router
}

func (r *fiberRouter) Group(relativePath string, handlers ...any) Router {
	fiberHandlers := make([]fiber.Handler, len(handlers))
	for i, h := range handlers {
		fiberHandlers[i] = h.(fiber.Handler)
	}
	return &fiberRouter{group: r.group.Group(relativePath, fiberHandlers...)}
}

func (r *fiberRouter) Handle(method, path string, handlers ...any) {
	if len(handlers) == 0 {
		return
	}
	h := handlers[0].(fiber.Handler)
	r.group.Add(method, path, h)
}

func (r *fiberRouter) Use(handlers ...any) {
	fiberHandlers := make([]any, len(handlers))
	for i, h := range handlers {
		fiberHandlers[i] = h
	}
	r.group.Use(fiberHandlers...)
}

func initFiber(app *fiber.App, prefix string) error {
	currentFramework = frameworkFiber
	g := mmfiber.Init(app, prefix)
	currentGroup = &fiberRouter{group: g}
	return nil
}

func fiberDo[T any](path string, method string, handler Handler[T]) {
	rg := currentGroup.(*fiberRouter)
	rg.group.Add(method, path, func(c *fiber.Ctx) error {
		body := c.Body()
		var req T
		if err := decodeBody(body, &req); err != nil {
			return fiberAbort(c, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
		}
		httpReq := &http.Request{Method: c.Method(), Header: make(http.Header)}
		data, err := handler(httpReq, &req)
		if err != nil {
			return fiberAbort(c, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
		}
		encoded, err := mm.EncodeFromValue(data, "")
		if err != nil {
			return fiberAbort(c, http.StatusInternalServerError, mmError{Error: "encode failed"})
		}
		c.Set("Content-Type", ContentTypeMetaMessage)
		return c.Status(http.StatusOK).Send(encoded)
	})
	rg.group.Options(path, func(c *fiber.Ctx) error {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			return fiberAbort(c, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
		}
		c.Set("Allow", method+", OPTIONS")
		c.Set("Content-Type", ContentTypeMetaMessage)
		return c.Status(http.StatusOK).Send(encoded)
	})
}

func fiberAbort(c *fiber.Ctx, code int, obj any) error {
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("encode error")
	}
	c.Set("Content-Type", ContentTypeMetaMessage)
	return c.Status(code).Send(encoded)
}

// ---------------------------------------------------------------------------
// Chi adapter
// ---------------------------------------------------------------------------

type chiRouter struct {
	group chi.Router
}

func (r *chiRouter) Group(relativePath string, handlers ...any) Router {
	chiHandlers := make([]func(http.Handler) http.Handler, len(handlers))
	for i, h := range handlers {
		chiHandlers[i] = h.(func(http.Handler) http.Handler)
	}
	rg := chi.NewRouter()
	rg.Use(chiHandlers...)
	r.group.Mount(relativePath, rg)
	return &chiRouter{group: rg}
}

func (r *chiRouter) Handle(method, path string, handlers ...any) {
	if len(handlers) == 0 {
		return
	}
	h := handlers[0].(http.HandlerFunc)
	r.group.Method(method, path, h)
}

func (r *chiRouter) Use(handlers ...any) {
	chiHandlers := make([]func(http.Handler) http.Handler, len(handlers))
	for i, h := range handlers {
		chiHandlers[i] = h.(func(http.Handler) http.Handler)
	}
	r.group.Use(chiHandlers...)
}

func initChi(r chi.Router, prefix string) error {
	currentFramework = frameworkChi
	rg := mmchi.Init(r, prefix)
	currentGroup = &chiRouter{group: rg}
	return nil
}

func chiDo[T any](path string, method string, handler Handler[T]) {
	rg := currentGroup.(*chiRouter)
	rg.group.Method(method, path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			chiAbort(w, r, http.StatusBadRequest, mmError{Error: "read body failed: " + err.Error()})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		var req T
		if err := decodeBody(body, &req); err != nil {
			chiAbort(w, r, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			chiAbort(w, r, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		encoded, err := mm.EncodeFromValue(data, "")
		if err != nil {
			chiAbort(w, r, http.StatusInternalServerError, mmError{Error: "encode failed"})
			return
		}
		w.Header().Set("Content-Type", ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	}))
	rg.group.Options(path, func(w http.ResponseWriter, r *http.Request) {
		var sample T
		encoded, err := mm.EncodeFromValue(sample, "example")
		if err != nil {
			chiAbort(w, r, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
			return
		}
		w.Header().Set("Allow", method+", OPTIONS")
		w.Header().Set("Content-Type", ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func chiAbort(w http.ResponseWriter, r *http.Request, code int, obj any) {
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", ContentTypeMetaMessage)
	w.WriteHeader(code)
	w.Write(encoded)
}

// ---------------------------------------------------------------------------
// Vanilla (net/http) adapter
// ---------------------------------------------------------------------------

type vanillaRouter struct {
	mux    *http.ServeMux
	prefix string
}

func (r *vanillaRouter) Group(relativePath string, handlers ...any) Router {
	return &vanillaRouter{
		mux:    r.mux,
		prefix: r.prefix + relativePath,
	}
}

func (r *vanillaRouter) Handle(method, path string, handlers ...any) {
	if len(handlers) == 0 {
		return
	}
	h := handlers[0].(http.HandlerFunc)
	fullPath := r.prefix + path
	r.mux.HandleFunc(fullPath, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != method {
			vanillaAbort(w, req, http.StatusMethodNotAllowed, mmError{Error: "method not allowed"})
			return
		}
		h(w, req)
	})
}

func (r *vanillaRouter) Use(handlers ...any) {
	// Vanilla net/http does not support middleware natively.
	// For middleware support, use a framework that supports it (Gin, Echo, Fiber, Chi).
}

func initVanilla(mux *http.ServeMux, prefix string) error {
	currentFramework = frameworkVanilla
	mmvanilla.Init(mux, prefix)
	currentGroup = &vanillaRouter{mux: mux, prefix: prefix}
	return nil
}

func vanillaDo[T any](path string, method string, handler Handler[T]) {
	rg := currentGroup.(*vanillaRouter)
	fullPath := rg.prefix + path
	rg.mux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			var sample T
			encoded, err := mm.EncodeFromValue(sample, "example")
			if err != nil {
				vanillaAbort(w, r, http.StatusInternalServerError, mmError{Error: "schema encode failed"})
				return
			}
			w.Header().Set("Allow", method+", OPTIONS")
			w.Header().Set("Content-Type", ContentTypeMetaMessage)
			w.WriteHeader(http.StatusOK)
			w.Write(encoded)
			return
		}
		if r.Method != method {
			vanillaAbort(w, r, http.StatusMethodNotAllowed, mmError{Error: "method not allowed"})
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			vanillaAbort(w, r, http.StatusBadRequest, mmError{Error: "read body failed: " + err.Error()})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		var req T
		if err := decodeBody(body, &req); err != nil {
			vanillaAbort(w, r, http.StatusBadRequest, mmError{Error: "bind failed: " + err.Error()})
			return
		}
		data, err := handler(r, &req)
		if err != nil {
			vanillaAbort(w, r, http.StatusUnprocessableEntity, mmError{Error: err.Error()})
			return
		}
		encoded, err := mm.EncodeFromValue(data, "")
		if err != nil {
			vanillaAbort(w, r, http.StatusInternalServerError, mmError{Error: "encode failed"})
			return
		}
		w.Header().Set("Content-Type", ContentTypeMetaMessage)
		w.WriteHeader(http.StatusOK)
		w.Write(encoded)
	})
}

func vanillaAbort(w http.ResponseWriter, r *http.Request, code int, obj any) {
	encoded, err := mm.EncodeFromValue(obj, "")
	if err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", ContentTypeMetaMessage)
	w.WriteHeader(code)
	w.Write(encoded)
}

// ---------------------------------------------------------------------------
// Package-level route registration functions
// ---------------------------------------------------------------------------

// GET registers a GET route on the active route group.
// handlers are framework-specific types (gin.HandlerFunc, echo.HandlerFunc, etc.).
func GET(relativePath string, handlers ...any) {
	switch currentFramework {
	case frameworkGin:
		ginHandlers := make([]gin.HandlerFunc, len(handlers))
		for i, h := range handlers {
			ginHandlers[i] = h.(gin.HandlerFunc)
		}
		mmgin.GET(relativePath, ginHandlers...)
	case frameworkEcho:
		if len(handlers) == 0 {
			return
		}
		mmecho.GET(relativePath, handlers[0].(echo.HandlerFunc))
	case frameworkFiber:
		if len(handlers) == 0 {
			return
		}
		mmfiber.GET(relativePath, handlers[0].(fiber.Handler))
	case frameworkChi:
		if len(handlers) == 0 {
			return
		}
		mmchi.GET(relativePath, handlers[0].(http.HandlerFunc))
	case frameworkVanilla:
		vanillaHandlers := make([]http.HandlerFunc, len(handlers))
		for i, h := range handlers {
			vanillaHandlers[i] = h.(http.HandlerFunc)
		}
		mmvanilla.GET(relativePath, vanillaHandlers...)
	}
}

// HEAD registers a HEAD route on the active route group.
func HEAD(relativePath string, handlers ...any) {
	switch currentFramework {
	case frameworkGin:
		ginHandlers := make([]gin.HandlerFunc, len(handlers))
		for i, h := range handlers {
			ginHandlers[i] = h.(gin.HandlerFunc)
		}
		mmgin.HEAD(relativePath, ginHandlers...)
	case frameworkEcho:
		if len(handlers) == 0 {
			return
		}
		mmecho.HEAD(relativePath, handlers[0].(echo.HandlerFunc))
	case frameworkFiber:
		if len(handlers) == 0 {
			return
		}
		mmfiber.HEAD(relativePath, handlers[0].(fiber.Handler))
	case frameworkChi:
		if len(handlers) == 0 {
			return
		}
		mmchi.HEAD(relativePath, handlers[0].(http.HandlerFunc))
	case frameworkVanilla:
		vanillaHandlers := make([]http.HandlerFunc, len(handlers))
		for i, h := range handlers {
			vanillaHandlers[i] = h.(http.HandlerFunc)
		}
		mmvanilla.HEAD(relativePath, vanillaHandlers...)
	}
}

// DELETE registers a DELETE route on the active route group.
func DELETE(relativePath string, handlers ...any) {
	switch currentFramework {
	case frameworkGin:
		ginHandlers := make([]gin.HandlerFunc, len(handlers))
		for i, h := range handlers {
			ginHandlers[i] = h.(gin.HandlerFunc)
		}
		mmgin.DELETE(relativePath, ginHandlers...)
	case frameworkEcho:
		if len(handlers) == 0 {
			return
		}
		mmecho.DELETE(relativePath, handlers[0].(echo.HandlerFunc))
	case frameworkFiber:
		if len(handlers) == 0 {
			return
		}
		mmfiber.DELETE(relativePath, handlers[0].(fiber.Handler))
	case frameworkChi:
		if len(handlers) == 0 {
			return
		}
		mmchi.DELETE(relativePath, handlers[0].(http.HandlerFunc))
	case frameworkVanilla:
		vanillaHandlers := make([]http.HandlerFunc, len(handlers))
		for i, h := range handlers {
			vanillaHandlers[i] = h.(http.HandlerFunc)
		}
		mmvanilla.DELETE(relativePath, vanillaHandlers...)
	}
}

// OPTIONS registers an OPTIONS route on the active route group.
func OPTIONS(relativePath string, handlers ...any) {
	switch currentFramework {
	case frameworkGin:
		ginHandlers := make([]gin.HandlerFunc, len(handlers))
		for i, h := range handlers {
			ginHandlers[i] = h.(gin.HandlerFunc)
		}
		mmgin.OPTIONS(relativePath, ginHandlers...)
	case frameworkEcho:
		if len(handlers) == 0 {
			return
		}
		mmecho.OPTIONS(relativePath, handlers[0].(echo.HandlerFunc))
	case frameworkFiber:
		if len(handlers) == 0 {
			return
		}
		mmfiber.OPTIONS(relativePath, handlers[0].(fiber.Handler))
	case frameworkChi:
		if len(handlers) == 0 {
			return
		}
		mmchi.OPTIONS(relativePath, handlers[0].(http.HandlerFunc))
	case frameworkVanilla:
		vanillaHandlers := make([]http.HandlerFunc, len(handlers))
		for i, h := range handlers {
			vanillaHandlers[i] = h.(http.HandlerFunc)
		}
		mmvanilla.OPTIONS(relativePath, vanillaHandlers...)
	}
}

// Any registers a route for all HTTP methods on the active route group.
func Any(relativePath string, handlers ...any) {
	switch currentFramework {
	case frameworkGin:
		ginHandlers := make([]gin.HandlerFunc, len(handlers))
		for i, h := range handlers {
			ginHandlers[i] = h.(gin.HandlerFunc)
		}
		mmgin.Any(relativePath, ginHandlers...)
	case frameworkEcho:
		if len(handlers) == 0 {
			return
		}
		mmecho.Any(relativePath, handlers[0].(echo.HandlerFunc))
	case frameworkFiber:
		if len(handlers) == 0 {
			return
		}
		mmfiber.Any(relativePath, handlers[0].(fiber.Handler))
	case frameworkChi:
		if len(handlers) == 0 {
			return
		}
		mmchi.Any(relativePath, handlers[0].(http.HandlerFunc))
	case frameworkVanilla:
		vanillaHandlers := make([]http.HandlerFunc, len(handlers))
		for i, h := range handlers {
			vanillaHandlers[i] = h.(http.HandlerFunc)
		}
		mmvanilla.Any(relativePath, vanillaHandlers...)
	}
}

// POST registers a POST route with automatic request binding and
// an OPTIONS route on the same path for schema discovery.
// handler receives a framework-agnostic *http.Request and *T.
func POST[T any](relativePath string, handler Handler[T]) {
	switch currentFramework {
	case frameworkGin:
		ginDo(relativePath, "POST", handler)
	case frameworkEcho:
		echoDo(relativePath, "POST", handler)
	case frameworkFiber:
		fiberDo(relativePath, "POST", handler)
	case frameworkChi:
		chiDo(relativePath, "POST", handler)
	case frameworkVanilla:
		vanillaDo(relativePath, "POST", handler)
	}
}

// PUT registers a PUT route with automatic request binding and
// an OPTIONS route on the same path for schema discovery.
func PUT[T any](relativePath string, handler Handler[T]) {
	switch currentFramework {
	case frameworkGin:
		ginDo(relativePath, "PUT", handler)
	case frameworkEcho:
		echoDo(relativePath, "PUT", handler)
	case frameworkFiber:
		fiberDo(relativePath, "PUT", handler)
	case frameworkChi:
		chiDo(relativePath, "PUT", handler)
	case frameworkVanilla:
		vanillaDo(relativePath, "PUT", handler)
	}
}

// PATCH registers a PATCH route with automatic request binding and
// an OPTIONS route on the same path for schema discovery.
func PATCH[T any](relativePath string, handler Handler[T]) {
	switch currentFramework {
	case frameworkGin:
		ginDo(relativePath, "PATCH", handler)
	case frameworkEcho:
		echoDo(relativePath, "PATCH", handler)
	case frameworkFiber:
		fiberDo(relativePath, "PATCH", handler)
	case frameworkChi:
		chiDo(relativePath, "PATCH", handler)
	case frameworkVanilla:
		vanillaDo(relativePath, "PATCH", handler)
	}
}
