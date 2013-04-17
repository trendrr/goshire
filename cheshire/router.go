package cheshire

import (
	"log"
	"path"
	"strings"
	"sync"
)

// This is a default implementation of a cheshire Router.  
// it is based on the HTTP request multiplexer from golang sources:
// http://golang.org/src/pkg/net/http/server.go?s=25470:25535#L841
// 
// Patterns name fixed, rooted paths, like "/favicon.ico",
// or rooted subtrees, like "/images/" (note the trailing slash).
// Longer patterns take precedence over shorter ones, so that
// if there are handlers registered for both "/images/"
// and "/images/thumbnails/", the latter handler will be
// called for paths beginning "/images/thumbnails/" and the
// former will receive requests for any other paths in the
// "/images/" subtree.
//
// Should also takes care of sanitizing the URL request path,
// redirecting any request containing . or .. elements to an
// equivalent .- and ..-free URL.
type Router struct {
	mu              sync.RWMutex
	gets            map[string]muxEntry
	posts           map[string]muxEntry
	deletes         map[string]muxEntry
	puts            map[string]muxEntry
	NotFoundHandler Controller
}

type muxEntry struct {
	explicit bool
	h        Controller
	pattern  string
}

type DefaultNotFoundHandler struct {
}

func (h *DefaultNotFoundHandler) Config() *ControllerConfig {
	return nil
}
func (h *DefaultNotFoundHandler) HandleRequest(req *Request, conn Writer) {
	response := NewResponse(req)
	response.SetStatusCode(404)
	response.SetStatusMessage("Not Found")
	conn.Write(response)
}

// NewServeMux allocates and returns a new CheshireMux.
func NewDefaultRouter() *Router {
	router := &Router{gets: make(map[string]muxEntry),
		posts:   make(map[string]muxEntry),
		deletes: make(map[string]muxEntry),
		puts:    make(map[string]muxEntry)}
	router.NotFoundHandler = new(DefaultNotFoundHandler)
	return router
}

// Does path match pattern?
func pathMatch(pattern, path string) bool {
	if len(pattern) == 0 {
		// should not happen
		return false
	}
	n := len(pattern)
	if pattern[n-1] != '/' {
		return pattern == path
	}
	return len(path) >= n && path[0:n] == pattern
}

// Return the canonical path for p, eliminating . and .. elements.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

// Find a handler on a handler map given a path string
// Most-specific (longest) pattern wins
func (this *Router) match(method string, path string) Controller {
	var h Controller
	var n = 0
	m, ok := this.getMethodMap(method)
	if !ok {
		return nil
	}

	for k, v := range m {
		if !pathMatch(k, path) {
			continue
		}
		if h == nil || len(k) > n {
			n = len(k)
			h = v.h
		}
	}
	return h
}

// Match returns the registered Controller that matches the
// request or, if no match the registered not found handler is returned
func (mux *Router) Match(method string, path string) (h Controller) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	h = mux.match(method, path)

	if h == nil {
		log.Print("Not Found.  TODO: do something!")
		h = mux.NotFoundHandler
	}
	return
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (this *Router) Register(methods []string, handler Controller) {
	this.mu.Lock()
	defer this.mu.Unlock()
	for _, m := range methods {
		this.reg(m, handler)
	}
}

func (this *Router) getMethodMap(method string) (map[string]muxEntry, bool) {
	m := strings.ToUpper(method)
	switch m {
	case "GET":
		return this.gets, true
	case "POST":
		return this.posts, true
	case "PUT":
		return this.puts, true
	case "DELETE":
		return this.deletes, true
	}
	return nil, false
}

func (this *Router) reg(method string, handler Controller) {
	var pattern = handler.Config().Route
	m, ok := this.getMethodMap(method)

	if !ok {
		panic("cheshire: " + method + " is not a valid method!")
	}

	if pattern == "" {
		panic("cheshire: invalid pattern " + pattern)
	}
	if handler == nil {
		panic("cheshire: nil handler")
	}
	if m[pattern].explicit {
		panic("cheshire: multiple registrations for " + pattern)
	}

	m[pattern] = muxEntry{explicit: true, h: handler, pattern: pattern}
}
