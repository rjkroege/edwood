// Package sync provides centralized locking utilities for edwood.
package sync

// RefCounted represents a type with reference counting, like Window.
type RefCounted interface {
	Locker
	Inc()
	Dec() int
	SetOwner(owner int)
	Owner() int
	Close()
}

// WindowLocker provides a type-safe wrapper for Window locking operations.
// It handles reference counting and owner tracking as required by Window.Lock/Unlock.
type WindowLocker struct {
	target RefCounted
	locked bool
}

// NewWindowLocker creates a new WindowLocker wrapping the given RefCounted.
func NewWindowLocker(rc RefCounted) *WindowLocker {
	return &WindowLocker{target: rc}
}

// Lock acquires the window lock, increments the reference count, and sets the owner.
// The owner parameter identifies who holds the lock:
//   - 'K' - Keyboard thread
//   - 'E' - Error window operations
//   - 'M' - Mouse thread
//   - Command rune - Edit commands
func (wl *WindowLocker) Lock(owner int) {
	wl.target.Lock()
	wl.target.Inc()
	wl.target.SetOwner(owner)
	wl.locked = true
}

// Unlock releases the window lock, clears the owner, and decrements the reference count.
// If the reference count reaches zero, Close is called.
func (wl *WindowLocker) Unlock() {
	wl.target.SetOwner(0)
	if wl.target.Dec() == 0 {
		wl.target.Close()
	}
	wl.locked = false
	wl.target.Unlock()
}

// IsLocked returns true if the lock is currently held.
func (wl *WindowLocker) IsLocked() bool {
	return wl.locked
}

// WithLock executes fn while holding the lock, ensuring unlock on return.
// The lock is released even if fn panics.
func (wl *WindowLocker) WithLock(owner int, fn func()) {
	wl.Lock(owner)
	defer wl.Unlock()
	fn()
}
