package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ==========================================
// 1. CIRCUIT BREAKER CORE LOGIC
// ==========================================

// State represents the current status of the circuit breaker
type State int

const (
	StateClosed   State = iota // Normal: Traffic flows
	StateOpen                  // Tripped: Traffic blocked immediately
	StateHalfOpen              // Testing: One request allowed through
)

var ErrCircuitOpen = errors.New("circuit breaker is open: partner API is temporarily down")

// CircuitBreaker protects our system from failing 3rd-party dependencies
type CircuitBreaker struct {
	mu                  sync.RWMutex
	state               State
	failureThreshold    int
	consecutiveFailures int
	resetTimeout        time.Duration
	lastFailureTime     time.Time
}

// NewCircuitBreaker initializes the breaker with your chosen rules
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: threshold,
		resetTimeout:     timeout,
	}
}

// Execute is the wrapper around the flaky external API call
func (cb *CircuitBreaker) Execute(operation func() (string, error)) (string, error) {
	// 1. Check if we should even try to make the request
	if !cb.canAttempt() {
		return "", ErrCircuitOpen
	}

	// 2. Actually call the external API (Stripe, in this case)
	result, err := operation()

	// 3. Update the state based on the result
	if err != nil {
		cb.recordFailure()
		return "", err
	}

	cb.recordSuccess()
	return result, nil
}

// canAttempt determines if the request should be blocked or allowed
func (cb *CircuitBreaker) canAttempt() bool {
	cb.mu.RLock()
	state := cb.state
	lastFailure := cb.lastFailureTime
	cb.mu.RUnlock()

	if state == StateClosed {
		return true
	}

	if state == StateOpen {
		// If enough time has passed, transition to Half-Open to test the waters
		if time.Since(lastFailure) > cb.resetTimeout {
			cb.mu.Lock()
			if cb.state == StateOpen { // Double-check to prevent race conditions
				cb.state = StateHalfOpen
			}
			cb.mu.Unlock()
			return true // Let this ONE test request through
		}
		return false // Still open, block the request
	}

	// If Half-Open, we are currently testing. Allow it.
	return state == StateHalfOpen
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFailures++
	cb.lastFailureTime = time.Now()

	// Trip the breaker if we hit the limit, OR if our Half-Open test failed
	if cb.state == StateHalfOpen || cb.consecutiveFailures >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// The API is healthy again! Reset everything.
	cb.consecutiveFailures = 0
	cb.state = StateClosed
}

// ==========================================
// 2. FINTECH SIMULATION (STRIPE PAYOUTS)
// ==========================================

// Simulate Stripe servers being down initially
var stripeIsDown = true

// mockStripePayoutAPI represents a flaky 3rd-party banking provider
func mockStripePayoutAPI() (string, error) {
	time.Sleep(100 * time.Millisecond) // Simulate network latency

	if stripeIsDown {
		return "", fmt.Errorf("503 Service Unavailable: Stripe is currently down")
	}
	return "tx_success_987654321", nil
}

func main() {
	// Initialize breaker: Trip after 2 failures, wait 2 seconds before testing recovery
	cb := NewCircuitBreaker(2, 2*time.Second)

	fmt.Println("--- STARTING DISBURSEMENTS (STRIPE IS DOWN) ---")
	
	for i := 1; i <= 5; i++ {
		fmt.Printf("User %d requesting payout...\n", i)
		
		txID, err := cb.Execute(mockStripePayoutAPI)
		
		if err != nil {
			if errors.Is(err, ErrCircuitOpen) {
				// The Business Value: Fast failure, no hanging threads, graceful UI message
				fmt.Println("   -> [CIRCUIT OPEN] 🚦 Blocked instantly! Telling user: 'Payout queued due to partner delays.'")
			} else {
				fmt.Printf("   -> [API FAILED] ❌ Stripe returned an error: %v\n", err)
			}
		} else {
			fmt.Printf("   -> [SUCCESS] ✅ Payout complete! TX ID: %s\n", txID)
		}
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Println("\n--- WAITING 2 SECONDS (STRIPE ENGINEERS FIXING SERVERS) ---")
	time.Sleep(2 * time.Second)
	
	// Simulate Stripe fixing their outage
	stripeIsDown = false 

	fmt.Println("\n--- RESUMING DISBURSEMENTS (STRIPE IS UP) ---")
	for i := 6; i <= 8; i++ {
		fmt.Printf("User %d requesting payout...\n", i)
		
		txID, err := cb.Execute(mockStripePayoutAPI)
		
		if err != nil {
			fmt.Printf("   -> [ERROR] %v\n", err)
		} else {
			fmt.Printf("   -> [SUCCESS] ✅ Payout complete! TX ID: %s\n", txID)
		}
		time.Sleep(300 * time.Millisecond)
	}
}