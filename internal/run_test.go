package internal_test

import (
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"murmapp.hook/internal"
)

// TestRunGracefulShutdown simulates application startup and graceful shutdown
func TestRunGracefulShutdown(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	// Trigger shutdown after a short delay
	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(syscall.SIGTERM)
	}()

	err := internal.Run()
	require.NoError(t, err)
	wg.Wait()
}
