package japi

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/jarrettv/go-japi/problem"
)

type Empty struct{}

const JsonEncoding = "application/json"

type Middleware func(http.Handler) http.Handler

type Router interface {
	Get(path string, handle http.Handler)
	Post(path string, handle http.Handler)
	Put(path string, handle http.Handler)
	Delete(path string, handle http.Handler)
	Handle(method, path string, handle http.Handler)
	HandleFunc(method, path string, handle http.HandlerFunc)
	Group(path string) Router
	Use(mw ...Middleware)
}

type API struct {
	router *httprouter.Router
	config *Config
	mw     []Middleware

	NotFound         http.Handler
	MethodNotAllowed http.Handler
	PanicHandler     func(http.ResponseWriter, *http.Request, interface{})
}

// New creates a new API instance.
func New(c *Config) *API {
	if c == nil {
		c = GetDefaultConfig()
	}

	r := httprouter.New()
	r.RedirectTrailingSlash = true
	r.SaveMatchedRoutePath = true

	return &API{
		router:           r,
		config:           c,
		NotFound:         withConfig(E(problem.NotFound()), c),
		MethodNotAllowed: withConfig(E(problem.Status(http.StatusMethodNotAllowed)), c),
		PanicHandler: func(w http.ResponseWriter, r *http.Request, err any) {
			withConfig(E(problem.Status(http.StatusInternalServerError)), c).ServeHTTP(w, r)
		},
	}
}

// Router creates a http.Handler for the API.
func (r *API) Router() http.Handler {
	r.router.NotFound = r.NotFound
	r.router.MethodNotAllowed = r.MethodNotAllowed
	r.router.PanicHandler = r.PanicHandler
	r.router.SaveMatchedRoutePath = true

	h := http.Handler(r.router)
	// TODO (jv) understand why need to reverse middleware
	for i := len(r.mw) - 1; i >= 0; i-- {
		h = r.mw[i](h)
	}
	return h
}

// Get handles GET requests.
func (r *API) Get(path string, handle http.Handler) {
	r.Handle(http.MethodGet, path, handle)
}

// Post handles POST requests.
func (r *API) Post(path string, handle http.Handler) {
	r.Handle(http.MethodPost, path, handle)
}

// Put handles PUT requests.
func (r *API) Put(path string, handle http.Handler) {
	r.Handle(http.MethodPut, path, handle)
}

// Patch handles PATCH requests.
func (r *API) Patch(path string, handle http.Handler) {
	r.Handle(http.MethodPatch, path, handle)
}

// Delete handles DELETE requests.
func (r *API) Delete(path string, handle http.Handler) {
	r.Handle(http.MethodDelete, path, handle)
}

// Handle can be used to wrap regular handlers.
func (r *API) Handle(method, path string, handle http.Handler) {
	var hh httprouter.Handle
	if h, ok := handle.(Handler); ok {
		hh = withConfig(h, r.config).handle
	} else {
		hh = wrapHandler(handle)
	}

	r.router.Handle(method, path, hh)
}

// HandleFunc handles the requests with the specified method.
func (r *API) HandleFunc(method, path string, handle http.HandlerFunc) {
	r.Handle(method, path, handle)
}

// Group creates a new sub-router with the given prefix.
func (r *API) Group(path string) Router {
	return &group{prefix: path, r: r}
}

// Use will register middleware to run prior to the handlers.
func (r *API) Use(mw ...Middleware) {
	r.mw = append(r.mw, mw...)
}

func withConfig(handle Handler, c *Config) Handler {
	if h, ok := handle.(interface{ setConfig(*Config) }); ok {
		h.setConfig(c)
	}

	return handle
}

func wrapHandler(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		h.ServeHTTP(w, r)
	}
}
