package health

import (
	"fmt"
	"net/http"
	"sync"
)

// Check is a function that performs a health check.
type Check func() error

// Checker manages the health status of the application.
type Checker struct {
	mu              sync.RWMutex
	ready           bool
	readinessChecks map[string]Check
}

// NewChecker creates a new health checker.
func NewChecker() *Checker {
	return &Checker{
		readinessChecks: make(map[string]Check),
	}
}

// AddReadinessCheck adds a readiness check for a component.
func (c *Checker) AddReadinessCheck(name string, check Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readinessChecks[name] = check
}

// LivenessProbe is the liveness probe handler.
func (c *Checker) LivenessProbe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// ReadinessProbe is the readiness probe handler.
func (c *Checker) ReadinessProbe(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.ready {
		http.Error(w, "consumer not ready (no partitions assigned)", http.StatusServiceUnavailable)
		return
	}

	for name, check := range c.readinessChecks {
		if err := check(); err != nil {
			http.Error(w, fmt.Sprintf("%s is not ready: %v", name, err), http.StatusServiceUnavailable)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// IsReady returns the current readiness state.
func (c *Checker) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

// SetReady sets the readiness state.
func (c *Checker) SetReady(ready bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ready = ready
}
