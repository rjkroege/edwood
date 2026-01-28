// Package sync provides centralized locking utilities for edwood.
//
// This package offers type-safe, defer-friendly wrappers around the various
// locking patterns used throughout edwood, ensuring consistent lock ordering
// and proper unlock behavior.
//
// Lock Ordering: When acquiring multiple locks, always follow this order
// to prevent deadlocks:
//
//	Row lock
//	    └─► Window lock
//	           └─► Frame lock
package sync

// Locker is a basic interface for types that can be locked/unlocked.
type Locker interface {
	Lock()
	Unlock()
}

// RowLocker provides a type-safe wrapper for Row locking operations.
// It ensures consistent lock/unlock behavior and integrates with defer.
type RowLocker struct {
	locker Locker
	locked bool
}

// NewRowLocker creates a new RowLocker wrapping the given Locker.
func NewRowLocker(l Locker) *RowLocker {
	return &RowLocker{locker: l}
}

// Lock acquires the row lock.
func (rl *RowLocker) Lock() {
	rl.locker.Lock()
	rl.locked = true
}

// Unlock releases the row lock.
func (rl *RowLocker) Unlock() {
	rl.locked = false
	rl.locker.Unlock()
}

// IsLocked returns true if the lock is currently held.
func (rl *RowLocker) IsLocked() bool {
	return rl.locked
}

// WithLock executes fn while holding the lock, ensuring unlock on return.
// The lock is released even if fn panics.
func (rl *RowLocker) WithLock(fn func()) {
	rl.Lock()
	defer rl.Unlock()
	fn()
}
