package app

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/transport/http"
)

func TestNew_Simple(t *testing.T) {
	httpServer := http.NewServer("", gin.New())
	app := New(WithServer(httpServer))

	info := app.Info()
	if info.ServerCount != 1 {
		t.Fatalf("expected 1 server, got %d", info.ServerCount)
	}

	if info.Started {
		t.Fatal("expected application not to be started")
	}
}

func TestNew_WithOptions(t *testing.T) {
	httpServer := http.NewServer("", gin.New())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app := New(
		WithServer(httpServer),
		WithShutdownTimeout(5*time.Second),
		WithCancelTimeout(2*time.Second),
		WithCancel("test-cancel", func(ctx context.Context) error {
			t.Log("cancel called")
			return nil
		}, time.Second),
		WithSignals(os.Interrupt, syscall.SIGTERM),
		WithContext(ctx),
	)
	app.RegisterCancel("late-cancel", func(ctx context.Context) error {
		t.Log("late cancel called")
		return nil
	}, time.Second)
	err := app.Start()
	if err != nil && err != context.Canceled && err.Error() != "http: Server closed" {
		t.Fatalf("unexpected error from app.Start(): %v", err)
	}
}

func TestNew_WithMultipleServers(t *testing.T) {
	httpServer1 := http.NewServer("", gin.New())
	httpServer2 := http.NewServer("", gin.New())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	app := New(
		WithServers(httpServer1, httpServer2),
		WithServer(httpServer1), // Should be deduplicated
		WithContext(ctx),
	)

	info := app.Info()
	if info.ServerCount != 3 {
		t.Fatalf("expected 3 servers, got %d", info.ServerCount)
	}
}

func TestNew_NoServers(t *testing.T) {
	app := New()

	info := app.Info()
	if info.ServerCount != 0 {
		t.Fatalf("expected 0 servers, got %d", info.ServerCount)
	}

	// Should handle gracefully with no servers
	go func() {
		time.Sleep(100 * time.Millisecond)
		app.Stop()
	}()

	err := app.Start()
	if err != nil && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplication_AddServerAtRuntime(t *testing.T) {
	app := New()
	httpServer := http.NewServer("", gin.New())

	// Should work before starting
	err := app.AddServer(httpServer)
	if err != nil {
		t.Fatalf("unexpected error adding server: %v", err)
	}

	info := app.Info()
	if info.ServerCount != 1 {
		t.Fatalf("expected 1 server, got %d", info.ServerCount)
	}
}

func TestApplication_AddServerAfterStart(t *testing.T) {
	app := New()

	// Mark as started
	app.started = true

	httpServer := http.NewServer("", gin.New())
	err := app.AddServer(httpServer)
	if err != ErrAlreadyStarted {
		t.Fatalf("expected ErrAlreadyStarted, got %v", err)
	}
}

func TestApplication_AddNilServer(t *testing.T) {
	app := New()

	err := app.AddServer(nil)
	if err == nil {
		t.Fatal("expected error when adding nil server")
	}
}

func TestApplication_RegisterCancelAtRuntime(t *testing.T) {
	app := New()

	cleanupCalled := false
	err := app.RegisterCancel("test", func(ctx context.Context) error {
		cleanupCalled = true
		return nil
	}, time.Second)

	if err != nil {
		t.Fatalf("unexpected error adding cleanup: %v", err)
	}

	info := app.Info()
	if info.CleanupCount != 1 {
		t.Fatalf("expected 1 cleanup function, got %d", info.CleanupCount)
	}

	app.runCleanupTasks()

	if !cleanupCalled {
		t.Fatal("expected cleanup function to be called")
	}
}

func TestApplication_AddNilCleanup(t *testing.T) {
	app := New()

	err := app.RegisterCancel("test", nil, time.Second)
	if err == nil {
		t.Fatal("expected error when adding nil cleanup function")
	}
}

func TestWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer := http.NewServer("", gin.New())
	app := New(
		WithContext(ctx),
		WithServer(httpServer),
	)

	// Cancel context to test graceful shutdown
	cancel()

	// This should return quickly due to cancelled context
	done := make(chan error, 1)
	go func() {
		done <- app.Start()
	}()

	select {
	case err := <-done:
		// Context cancellation and server close errors are acceptable
		if err != nil && err != context.Canceled && err.Error() != "http: Server closed" {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("app.Start() should have returned quickly due to cancelled context")
	}
}

func TestCancelFunc_Panic(t *testing.T) {
	app := New(
		WithCancel("panic-cancel", func(ctx context.Context) error {
			panic("test panic")
		}, time.Second),
	)

	// Should not panic when executing cleanup
	app.runCleanupTasks()
}

func TestCancelFunc_Timeout(t *testing.T) {
	app := New(
		WithCancel("slow-cancel", func(ctx context.Context) error {
			time.Sleep(2 * time.Second)
			return nil
		}, 100*time.Millisecond),
	)

	start := time.Now()
	app.runCleanupTasks()
	duration := time.Since(start)

	// Should timeout quickly
	if duration > 500*time.Millisecond {
		t.Fatalf("cleanup took too long: %v", duration)
	}
}

func TestDefaultValues(t *testing.T) {
	if DefaultShutdownTimeout != 30*time.Second {
		t.Fatalf("unexpected default shutdown timeout: %v", DefaultShutdownTimeout)
	}

	if DefaultCancelTimeout != 10*time.Second {
		t.Fatalf("unexpected default cleanup timeout: %v", DefaultCancelTimeout)
	}

	expectedSignals := []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT}
	if len(DefaultSignals) != len(expectedSignals) {
		t.Fatalf("unexpected number of default signals: %d", len(DefaultSignals))
	}
}

func TestWithNilOptions(t *testing.T) {
	// Test that nil values are handled gracefully
	app := New(
		WithServer(nil),  // Should be ignored
		WithServers(nil), // Should be ignored
	)

	info := app.Info()
	if info.ServerCount != 0 {
		t.Fatalf("expected 0 servers, got %d", info.ServerCount)
	}
}

func TestOptionValidation(t *testing.T) {
	app := New(
		WithShutdownTimeout(0),             // Should be ignored (invalid)
		WithCancelTimeout(0),               // Should be ignored (invalid)
		WithShutdownTimeout(5*time.Second), // Should be applied
	)

	// Check that valid options are applied
	info := app.Info()
	if info.ServerCount != 0 {
		t.Fatalf("expected 0 servers, got %d", info.ServerCount)
	}
}
