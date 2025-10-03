package netutil

import (
	"math"
	"math/rand"
	"time"
)

// JitterType defines the type of jitter to apply
type JitterType int

const (
	NoJitter JitterType = iota
	FullJitter
	EqualJitter
	DecorrelatedJitter
)

// BackoffConfig holds the configuration for the backoff state machine
type BackoffConfig struct {
	BaseDelay  time.Duration // Initial delay
	MaxDelay   time.Duration // Maximum delay cap
	Multiplier float64       // Exponential multiplier (typically 2.0)
	MaxRetries int           // Maximum number of retries (0 = unlimited)
	JitterType JitterType    // Type of jitter to apply
}

// Backoff represents the backoff state machine
type Backoff struct {
	config        BackoffConfig
	currentRetry  int
	currentDelay  time.Duration
	previousDelay time.Duration
	rng           *rand.Rand
}

// NewBackoff creates a new backoff state machine with the given configuration
func NewBackoff(
	config BackoffConfig,
	RandomSource *rand.Rand, // Custom random source (nil = use default)
) *Backoff {
	// Set defaults
	if config.BaseDelay == 0 {
		config.BaseDelay = 1 * time.Second
	}
	if config.MaxDelay == 0 {
		config.MaxDelay = 64 * time.Second
	}
	if config.Multiplier == 0 {
		config.Multiplier = 2.0
	}

	rng := RandomSource
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	return &Backoff{
		config:        config,
		currentRetry:  0,
		currentDelay:  0,
		previousDelay: config.BaseDelay,
		rng:           rng,
	}
}

// NextDelay calculates and returns the next backoff delay
// Returns the delay and a boolean indicating if more retries are allowed
func (sm *Backoff) NextDelay() (time.Duration, bool) {
	// Check if we've exceeded max retries
	if sm.config.MaxRetries > 0 && sm.currentRetry >= sm.config.MaxRetries {
		return 0, false
	}

	sm.currentRetry++

	// Calculate base exponential delay
	exponentialDelay := float64(sm.config.BaseDelay) * math.Pow(sm.config.Multiplier, float64(sm.currentRetry-1))

	// Apply max delay cap
	if exponentialDelay > float64(sm.config.MaxDelay) {
		exponentialDelay = float64(sm.config.MaxDelay)
	}

	// Apply jitter
	var delay time.Duration
	switch sm.config.JitterType {
	case NoJitter:
		delay = time.Duration(exponentialDelay)
	case FullJitter:
		delay = time.Duration(sm.rng.Float64() * exponentialDelay)
	case EqualJitter:
		halfDelay := exponentialDelay / 2
		delay = time.Duration(halfDelay + sm.rng.Float64()*halfDelay)
	case DecorrelatedJitter:
		// temp = min(cap, random(base, previous_delay * 3))
		temp := float64(sm.config.BaseDelay) + sm.rng.Float64()*(float64(sm.previousDelay)*3-float64(sm.config.BaseDelay))
		if temp > float64(sm.config.MaxDelay) {
			temp = float64(sm.config.MaxDelay)
		}
		delay = time.Duration(temp)
		sm.previousDelay = delay
	}

	sm.currentDelay = delay
	return delay, true
}

// Reset resets the state machine to its initial state
func (sm *Backoff) Reset() {
	sm.currentRetry = 0
	sm.currentDelay = 0
	sm.previousDelay = sm.config.BaseDelay
}

// RetryCount returns the current retry count
func (sm *Backoff) RetryCount() int {
	return sm.currentRetry
}

// CurrentDelay returns the most recently calculated delay
func (sm *Backoff) CurrentDelay() time.Duration {
	return sm.currentDelay
}

// CanRetry returns whether another retry is allowed
func (sm *Backoff) CanRetry() bool {
	if sm.config.MaxRetries == 0 {
		return true
	}
	return sm.currentRetry < sm.config.MaxRetries
}
