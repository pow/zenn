package graceful

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestReadyzFlipsTo503OnShutdown verifies that /readyz starts returning
// 503 as soon as Shutdown() is called, which is what lets a Kubernetes
// readiness probe remove the Pod from Service endpoints before new
// connections are refused.
func TestReadyzFlipsTo503OnShutdown(t *testing.T) {
	s := NewServer(":0", 0)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("pre-shutdown /readyz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 before shutdown, got %d", resp.StatusCode)
	}

	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	resp, err = http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("post-shutdown /readyz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 after shutdown, got %d", resp.StatusCode)
	}
}

// TestShutdownWaitsForInFlightRequest starts a slow request, calls
// Shutdown while the handler is still sleeping, and checks that the
// response completes with 200. This is the core guarantee we rely on
// during rolling updates: in-flight requests must not be cut off.
func TestShutdownWaitsForInFlightRequest(t *testing.T) {
	s := NewServer("127.0.0.1:0", 300*time.Millisecond)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.httpSrv.Serve(ln)
	}()

	url := "http://" + ln.Addr().String() + "/"

	// Fire the slow request.
	var (
		wg          sync.WaitGroup
		bodyBytes   []byte
		statusCode  int
		requestErr  error
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := http.Get(url)
		if err != nil {
			requestErr = err
			return
		}
		defer resp.Body.Close()
		statusCode = resp.StatusCode
		bodyBytes, requestErr = io.ReadAll(resp.Body)
	}()

	// Give the handler a moment to enter its sleep.
	time.Sleep(50 * time.Millisecond)

	// Trigger shutdown while the request is in-flight.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	wg.Wait()
	if requestErr != nil {
		t.Fatalf("in-flight request failed: %v", requestErr)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected 200 from in-flight request, got %d", statusCode)
	}
	if string(bodyBytes) != "done" {
		t.Fatalf("unexpected body: %q", bodyBytes)
	}

	if err := <-serveErr; err != nil && err != http.ErrServerClosed {
		t.Fatalf("serve returned unexpected error: %v", err)
	}
}

// TestShutdownRejectsNewConnections confirms that once Shutdown completes,
// a fresh connection to the listener is refused. This proves we are not
// relying on the kernel to drop packets — http.Server actively stops
// accepting new sockets.
func TestShutdownRejectsNewConnections(t *testing.T) {
	s := NewServer("127.0.0.1:0", 0)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.httpSrv.Serve(ln)
	}()

	// Warm up: one successful request.
	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("warm-up request: %v", err)
	}
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	// After Shutdown, a fresh request must not succeed. It may fail at
	// connect time or at read time depending on timing; either is fine.
	client := &http.Client{Timeout: 500 * time.Millisecond}
	if _, err := client.Get("http://" + addr + "/"); err == nil {
		t.Fatalf("expected error on post-shutdown request, got nil")
	}

	if err := <-serveErr; err != nil && err != http.ErrServerClosed {
		t.Fatalf("serve returned unexpected error: %v", err)
	}
}

// TestIsReadyReflectsShutdownState is a sanity check for the flag we read
// from the readiness handler.
func TestIsReadyReflectsShutdownState(t *testing.T) {
	s := NewServer(":0", 0)
	if !s.IsReady() {
		t.Fatalf("new server should be ready")
	}
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if s.IsReady() {
		t.Fatalf("server should not be ready after shutdown")
	}
}
