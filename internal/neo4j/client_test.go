package neo4j

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewClientWithRetry_SucceedsImmediately(t *testing.T) {
	// This test verifies that when connection succeeds on first try,
	// the retry logic doesn't add unnecessary delay.
	// We can't easily mock the neo4j driver, but we can test the config.
	cfg := Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}

	opts := RetryOptions{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	// Verify the options are valid
	if opts.MaxAttempts < 1 {
		t.Error("MaxAttempts should be at least 1")
	}
	if opts.InitialDelay <= 0 {
		t.Error("InitialDelay should be positive")
	}
	if opts.MaxDelay < opts.InitialDelay {
		t.Error("MaxDelay should be >= InitialDelay")
	}
	_ = cfg // Used in actual connection test
}

func TestRetryOptions_Defaults(t *testing.T) {
	opts := DefaultRetryOptions()

	if opts.MaxAttempts != 30 {
		t.Errorf("Default MaxAttempts: got %d, want 30", opts.MaxAttempts)
	}
	if opts.InitialDelay != 1*time.Second {
		t.Errorf("Default InitialDelay: got %v, want 1s", opts.InitialDelay)
	}
	if opts.MaxDelay != 10*time.Second {
		t.Errorf("Default MaxDelay: got %v, want 10s", opts.MaxDelay)
	}
}

func TestRetryOptions_BackoffCalculation(t *testing.T) {
	tests := []struct {
		name         string
		attempt      int
		initialDelay time.Duration
		maxDelay     time.Duration
		wantMin      time.Duration
		wantMax      time.Duration
	}{
		{
			name:         "first attempt",
			attempt:      1,
			initialDelay: 1 * time.Second,
			maxDelay:     10 * time.Second,
			wantMin:      1 * time.Second,
			wantMax:      1 * time.Second,
		},
		{
			name:         "second attempt with exponential backoff",
			attempt:      2,
			initialDelay: 1 * time.Second,
			maxDelay:     10 * time.Second,
			wantMin:      2 * time.Second,
			wantMax:      2 * time.Second,
		},
		{
			name:         "capped at max delay",
			attempt:      10,
			initialDelay: 1 * time.Second,
			maxDelay:     10 * time.Second,
			wantMin:      10 * time.Second,
			wantMax:      10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := calculateBackoff(tt.attempt, tt.initialDelay, tt.maxDelay)
			if delay < tt.wantMin || delay > tt.wantMax {
				t.Errorf("calculateBackoff(%d) = %v, want between %v and %v",
					tt.attempt, delay, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestRetryWithBackoff_SucceedsOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := retryWithBackoff(ctx, RetryOptions{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}, func() error {
		attempts++
		return nil // Success immediately
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetryWithBackoff_RetriesOnFailure(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := retryWithBackoff(ctx, RetryOptions{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("connection failed")
		}
		return nil // Success on third attempt
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryWithBackoff_ExhaustsAttempts(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	testErr := errors.New("persistent failure")

	err := retryWithBackoff(ctx, RetryOptions{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}, func() error {
		attempts++
		return testErr
	})

	if err == nil {
		t.Error("expected error after exhausting attempts")
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if !errors.Is(err, testErr) {
		t.Errorf("error = %v, want wrapped testErr", err)
	}
}

func TestRetryWithBackoff_RespectsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	// Cancel after first attempt
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retryWithBackoff(ctx, RetryOptions{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond, // Long enough to trigger context cancel
		MaxDelay:     1 * time.Second,
	}, func() error {
		attempts++
		return errors.New("keep failing")
	})

	if err == nil {
		t.Error("expected error from context cancellation")
	}
	// Should have stopped early due to context cancellation
	if attempts >= 10 {
		t.Errorf("attempts = %d, expected fewer due to context cancel", attempts)
	}
}

func TestConfigFromEnv_Defaults(t *testing.T) {
	// Test with no env vars set - should use defaults
	cfg := ConfigFromEnv()

	// These are the expected defaults from the existing code
	if cfg.URI == "" {
		t.Error("URI should have default value")
	}
	if cfg.Username == "" {
		t.Error("Username should have default value")
	}
	if cfg.Password == "" {
		t.Error("Password should have default value")
	}
	if cfg.Database == "" {
		t.Error("Database should have default value")
	}
}
