package api

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

// The Traefix features add their routes to the API here, so that the upstream handler only meets one line.
var (
	traefixRoutesMu sync.RWMutex
	traefixRoutes   = map[string]http.Handler{}
)

// AddTraefixRoute adds a GET route to the API, under its base path. The path starts with "/api/".
func AddTraefixRoute(path string, handler http.Handler) {
	traefixRoutesMu.Lock()
	defer traefixRoutesMu.Unlock()

	traefixRoutes[path] = handler
}

func appendTraefixRoutes(router *mux.Router) {
	traefixRoutesMu.RLock()
	defer traefixRoutesMu.RUnlock()

	for path, handler := range traefixRoutes {
		router.Methods(http.MethodGet).Path(path).Handler(handler)
	}
}
