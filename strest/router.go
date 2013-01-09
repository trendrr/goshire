package strest

import (
    "sync"
    "path"
    "log"
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
        mu    sync.RWMutex
        m     map[string]muxEntry
        NotFoundHandler Controller
}

type muxEntry struct {
        explicit bool
        h        Controller
        pattern  string
}


type DefaultNotFoundHandler struct {

}
func (h *DefaultNotFoundHandler) Config() (*Config) {
    return nil
}
func (h *DefaultNotFoundHandler) HandleRequest(req *Request, conn Connection) {
    response := NewResponse(req)
    response.SetStatusCode(404)
    response.SetStatusMessage("Not Found")
    conn.Write(response)
}

// NewServeMux allocates and returns a new CheshireMux.
func NewDefaultRouter() *Router { 
    router := &Router{m: make(map[string]muxEntry)} 
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
func (mux *Router) match(path string) Controller {
        var h Controller
        var n = 0
        for k, v := range mux.m {
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
// request or, 
//
// For now if no path matches the request, then nil is returned.
// in the future we might return a 404 controller 
func (mux *Router) Match(path string) (h Controller) {
        mux.mu.RLock()
        defer mux.mu.RUnlock()
        if h == nil {
                h = mux.match(path)
        }
        if h == nil {
                log.Print("Not Found.  TODO: do something!")
                h = mux.NotFoundHandler
        }
        return
}


// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (mux *Router) Register(handler Controller) {
        mux.mu.Lock()
        defer mux.mu.Unlock()
        var pattern = handler.Config().Route
        if pattern == "" {
                panic("cheshire: invalid pattern " + pattern)
        }
        if handler == nil {
                panic("cheshire: nil handler")
        }
        if mux.m[pattern].explicit {
                panic("cheshire: multiple registrations for " + pattern)
        }

        mux.m[pattern] = muxEntry{explicit: true, h: handler, pattern: pattern}


        // Helpful behavior:
        // If pattern is /tree/, insert an implicit permanent redirect for /tree.
        // It can be overridden by an explicit registration.
        // n := len(pattern)
        // if n > 0 && pattern[n-1] == '/' && !mux.m[pattern[0:n-1]].explicit {
        //         // If pattern contains a host name, strip it and use remaining
        //         // path for redirect.
        //         path := pattern
        //         if pattern[0] != '/' {
        //                 // In pattern, at least the last character is a '/', so
        //                 // strings.Index can't be -1.
        //                 path = pattern[strings.Index(pattern, "/"):]
        //         }
        //         //MOVED.  dont think we need this..
        //         // mux.m[pattern[0:n-1]] = muxEntry{h: RedirectHandler(path, StatusMovedPermanently), pattern: pattern}
        // }
}