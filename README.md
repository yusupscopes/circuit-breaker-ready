# 🚦 Go Circuit Breaker

A production-ready, thread-safe Circuit Breaker implementation in Go, designed to protect microservices from cascading failures when integrating with flaky third-party APIs (like payment gateways or KYC providers).

## 🏢 The Business Problem

In modern fintech, applications frequently rely on legacy banking infrastructure or third-party APIs (e.g., Stripe, Experian, Plaid). When these external partners experience downtime or extreme latency, it can cause **cascading failures** in your own system.

If 1,000 users attempt to process a payout while the Stripe API is hanging, your Go server will spawn 1,000 goroutines that get stuck waiting. This exhausts server memory and brings down your entire application—even the parts that don't rely on Stripe.

## 🛠️ The Solution

This project implements the **Circuit Breaker Pattern** to wrap external network calls. It monitors for consecutive failures and "trips" the circuit when a threshold is reached.

**Key Benefits:**

- **Fail-Fast:** Instantly rejects requests when the partner API is down, preventing resource exhaustion.
- **Graceful Degradation:** Allows the frontend to instantly show a polite "Delays expected" message rather than a spinning loading wheel that eventually times out.
- **Self-Healing:** Periodically lets a single test request through (Half-Open state) to check if the partner API has recovered, automatically resuming normal operations without human intervention.

## 🧠 State Machine Architecture

The breaker operates in three distinct states:

1. **Closed (Normal):** Requests flow freely. Failures are counted.
2. **Open (Tripped):** The failure threshold is met. All requests are blocked instantly and return an error (`ErrCircuitOpen`).
3. **Half-Open (Testing):** After a cooldown period, exactly _one_ request is allowed through to test the external API. If it succeeds, the circuit closes. If it fails, it trips back open.

## ✨ Technical Highlights

- **Thread-Safe:** Utilizes `sync.RWMutex` to ensure safe concurrent access across thousands of goroutines. Reads (checking state) are optimized with `RLock()`, while state mutations use full `Lock()`.
- **Zero Dependencies:** Built entirely using Go's standard library. No bloated external packages.
- **Clean State Management:** Uses `iota` for clear, readable state transitions.

## 🚀 How to Run the Simulation

This repository includes a simulation demonstrating a hypothetical Stripe Payouts outage and recovery.

```bash
go run main.go
```

**Simulation Output:**

1. The app attempts to process 5 payouts. The first two fail due to a simulated Stripe outage.
2. The Circuit Breaker **trips to Open**.
3. Requests 3, 4, and 5 are **blocked instantly** (protecting the app's resources).
4. A 2-second cooldown passes, and the Stripe engineers fix the outage.
5. Request 6 tests the waters (Half-Open), succeeds, and the breaker **recovers to Closed**.
6. Normal traffic resumes.
