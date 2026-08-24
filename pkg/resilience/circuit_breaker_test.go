package resilience

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_SuccessStayClosed(t *testing.T) {
	cb := NewCircuitBreaker(3, 50*time.Millisecond)

	err := cb.Execute(func() error {
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "CLOSED", cb.State())
	assert.Equal(t, 0, cb.Health()["failure_count"])
}

func TestCircuitBreaker_TripToOpenAndReject(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	dummyErr := errors.New("external call failed")

	// 1st failure
	err := cb.Execute(func() error { return dummyErr })
	assert.Equal(t, dummyErr, err)
	assert.Equal(t, "CLOSED", cb.State())

	// 2nd failure -> Trips to OPEN
	err = cb.Execute(func() error { return dummyErr })
	assert.Equal(t, dummyErr, err)
	assert.Equal(t, "OPEN", cb.State())

	// 3rd call immediately fails with ErrCircuitOpen without executing fn
	executed := false
	err = cb.Execute(func() error {
		executed = true
		return nil
	})
	assert.ErrorIs(t, err, ErrCircuitOpen)
	assert.False(t, executed)
}

func TestCircuitBreaker_HalfOpenRecoverySuccess(t *testing.T) {
	cb := NewCircuitBreaker(1, 20*time.Millisecond)
	dummyErr := errors.New("timeout")

	// Trip to OPEN
	_ = cb.Execute(func() error { return dummyErr })
	assert.Equal(t, "OPEN", cb.State())

	// Wait for recovery timeout
	time.Sleep(30 * time.Millisecond)

	// Execute should now try in HALF-OPEN and on success transition to CLOSED
	err := cb.Execute(func() error {
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "CLOSED", cb.State())
}

func TestCircuitBreaker_HalfOpenRecoveryFailure(t *testing.T) {
	cb := NewCircuitBreaker(1, 20*time.Millisecond)
	dummyErr := errors.New("timeout")

	// Trip to OPEN
	_ = cb.Execute(func() error { return dummyErr })
	assert.Equal(t, "OPEN", cb.State())

	// Wait for recovery timeout
	time.Sleep(30 * time.Millisecond)

	// Execute in HALF-OPEN fails -> Back to OPEN
	err := cb.Execute(func() error {
		return dummyErr
	})
	assert.Equal(t, dummyErr, err)
	assert.Equal(t, "OPEN", cb.State())
}
