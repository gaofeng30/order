package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/config"
)

func TestNewServerHasNonZeroTimeouts(t *testing.T) {
	server := newServer(":0", http.NotFoundHandler())
	if server.ReadHeaderTimeout <= 0 || server.ReadTimeout <= 0 || server.WriteTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Fatalf("server timeouts must be nonzero: %#v", server)
	}
}

func TestRunReturnsListenError(t *testing.T) {
	want := errors.New("address already in use")
	err := Run(context.Background(), testConfig(time.Second), http.NotFoundHandler(), discardLogger(), func(string, string) (net.Listener, error) {
		return nil, want
	})
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "listen") {
		t.Fatalf("Run() error = %v, want wrapped listen error", err)
	}
}

func TestRunStopsCleanly(t *testing.T) {
	listener := localListener(t)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- Run(ctx, testConfig(time.Second), http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusNoContent)
		}), discardLogger(), listenerOnce(listener))
	}()
	waitForStatus(t, listener.Addr().String(), http.StatusNoContent)

	cancel()
	if err := waitResult(t, result); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
}

func TestRunWaitsForInFlightRequest(t *testing.T) {
	listener := localListener(t)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- Run(ctx, testConfig(time.Second), http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			close(started)
			<-release
			writer.WriteHeader(http.StatusNoContent)
		}), discardLogger(), listenerOnce(listener))
	}()
	requestDone := make(chan error, 1)
	go func() {
		response, err := (&http.Client{Timeout: 2 * time.Second}).Get("http://" + listener.Addr().String())
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode != http.StatusNoContent {
				err = errors.New("unexpected response status")
			}
		}
		requestDone <- err
	}()
	waitChannel(t, started)

	cancel()
	select {
	case err := <-result:
		t.Fatalf("Run() returned before in-flight request completed: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	if err := waitResult(t, requestDone); err != nil {
		t.Fatalf("in-flight request error = %v", err)
	}
	if err := waitResult(t, result); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
}

func TestRunReturnsErrorWhenShutdownTimesOut(t *testing.T) {
	listener := localListener(t)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- Run(ctx, testConfig(40*time.Millisecond), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			close(started)
			<-release
		}), discardLogger(), listenerOnce(listener))
	}()
	requestDone := make(chan error, 1)
	go func() {
		response, err := (&http.Client{Timeout: time.Second}).Get("http://" + listener.Addr().String())
		if err == nil {
			_ = response.Body.Close()
		}
		requestDone <- err
	}()
	waitChannel(t, started)

	cancel()
	err := waitResult(t, result)
	if err == nil || !strings.Contains(err.Error(), "shutdown") {
		t.Fatalf("Run() error = %v, want shutdown timeout", err)
	}
	close(release)
	_ = waitResult(t, requestDone)
}

func localListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func listenerOnce(listener net.Listener) ListenFunc {
	return func(string, string) (net.Listener, error) {
		return listener, nil
	}
}

func testConfig(timeout time.Duration) config.Config {
	return config.Config{HTTPAddr: "127.0.0.1:0", ShutdownTimeout: timeout}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func waitForStatus(t *testing.T, address string, status int) {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get("http://" + address)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == status {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server at %s did not return %d", address, status)
}

func waitChannel(t *testing.T, channel <-chan struct{}) {
	t.Helper()
	select {
	case <-channel:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel")
	}
}

func waitResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for result")
		return nil
	}
}
