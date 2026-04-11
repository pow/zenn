// Package graceful provides a minimal HTTP server that demonstrates
// how to implement Kubernetes-aware graceful shutdown.
//
// The pattern:
//  1. /readyz returns 200 until Shutdown() is called, then returns 503.
//     This lets the Service drop the Pod from its Endpoints quickly.
//  2. / keeps serving in-flight requests until they complete.
//  3. http.Server.Shutdown stops accepting new connections and waits
//     for active handlers to finish, bounded by the caller's ctx.
package graceful

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"
)

// Server wraps http.Server with a readiness flag that flips to "not ready"
// as soon as shutdown begins.
type Server struct {
	httpSrv *http.Server
	// ready is set to 0 once Shutdown() is called. /readyz reads it.
	ready atomic.Bool
	// slowHandler simulates a long in-flight request; tests override this.
	slowHandlerDelay time.Duration
}

// NewServer returns a Server listening on addr. slowDelay controls how long
// the / handler sleeps before responding, which tests use to verify that
// Shutdown waits for in-flight requests.
func NewServer(addr string, slowDelay time.Duration) *Server {
	s := &Server{slowHandlerDelay: slowDelay}
	s.ready.Store(true)

	mux := http.NewServeMux()
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/", s.handleRoot)

	s.httpSrv = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

// Addr returns the underlying server's address (useful after ListenAndServe
// has bound a port when addr was ":0").
func (s *Server) Addr() string {
	return s.httpSrv.Addr
}

// Handler exposes the HTTP handler so tests can drive it with httptest.
func (s *Server) Handler() http.Handler {
	return s.httpSrv.Handler
}

// handleReadyz returns 200 when ready, 503 once shutdown has started.
// Kubernetes readiness probes should hit this so the Pod is removed from
// Service endpoints before http.Server.Shutdown stops accepting connections.
func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if s.ready.Load() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("shutting down"))
}

// handleRoot simulates an ordinary request handler. The delay exists so
// tests can start a request, trigger Shutdown mid-flight, and verify the
// response still completes successfully.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	select {
	case <-time.After(s.slowHandlerDelay):
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("done"))
	case <-r.Context().Done():
		// Client cancelled; nothing to do.
	}
}

// ListenAndServe is a thin wrapper around http.Server.ListenAndServe so
// callers can start the server without reaching into the wrapped field.
func (s *Server) ListenAndServe() error {
	err := s.httpSrv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown marks the server as not-ready and then delegates to
// http.Server.Shutdown, which refuses new connections and waits for active
// handlers to finish (bounded by ctx).
//
// Call order in production:
//  1. SIGTERM arrives (Kubernetes is terminating the Pod).
//  2. preStop sleep has already elapsed, giving kube-proxy time to drop
//     this Pod from Service endpoints.
//  3. Shutdown flips /readyz to 503 (belt-and-braces in case anything
//     still probes readiness) and drains in-flight requests.
func (s *Server) Shutdown(ctx context.Context) error {
	s.ready.Store(false)
	return s.httpSrv.Shutdown(ctx)
}

// IsReady reports whether the server is still accepting new work. Tests use
// it to avoid racing on the /readyz endpoint.
func (s *Server) IsReady() bool {
	return s.ready.Load()
}
