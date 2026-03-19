package main

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	// Setup: Threshold of 2 failures, very short 50ms timeout so tests run fast
	cb := NewCircuitBreaker(2, 50*time.Millisecond)

	// Mock operations for our tests to use
	successOp := func() (string, error) { return "success", nil }
	failOp := func() (string, error) { return "", errors.New("simulated API failure") }

	// --- TEST 1: Normal Operation ---
	t.Run("Normal Operation (Closed State)", func(t *testing.T) {
		res, err := cb.Execute(successOp)
		
		if err != nil || res != "success" {
			t.Fatalf("expected success, got error: %v", err)
		}
		if cb.state != StateClosed {
			t.Fatalf("expected state to be StateClosed, got %v", cb.state)
		}
	})

	// --- TEST 2: Tripping the Breaker ---
	t.Run("Tripping the Breaker (Closed -> Open)", func(t *testing.T) {
		// 1st failure (Doesn't trip yet)
		cb.Execute(failOp)
		if cb.state != StateClosed {
			t.Fatalf("expected state to remain StateClosed after 1 failure, got %v", cb.state)
		}

		// 2nd failure (Hits threshold of 2, trips to Open)
		cb.Execute(failOp)
		if cb.state != StateOpen {
			t.Fatalf("expected state to be StateOpen after 2 failures, got %v", cb.state)
		}

		// 3rd attempt: Even if the API is "healthy" now, the breaker should block it instantly
		_, err := cb.Execute(successOp) 
		if !errors.Is(err, ErrCircuitOpen) {
			t.Fatalf("expected ErrCircuitOpen, got %v", err)
		}
	})

	// --- TEST 3: Recovery ---
	t.Run("Recovery (Open -> Half-Open -> Closed)", func(t *testing.T) {
		// Wait for the 50ms reset timeout to pass
		time.Sleep(55 * time.Millisecond)

		// This attempt tests the waters (Half-Open). It succeeds, so it should close the circuit.
		res, err := cb.Execute(successOp)
		
		if err != nil || res != "success" {
			t.Fatalf("expected success on recovery, got error: %v", err)
		}
		if cb.state != StateClosed {
			t.Fatalf("expected state to reset to StateClosed, got %v", cb.state)
		}
		if cb.consecutiveFailures != 0 {
			t.Fatalf("expected failure count to reset to 0, got %d", cb.consecutiveFailures)
		}
	})
}