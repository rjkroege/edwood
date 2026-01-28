// Package sync provides centralized locking utilities for edwood.
//
// This package offers type-safe, defer-friendly wrappers around the various
// locking patterns used throughout edwood, ensuring consistent lock ordering
// and proper unlock behavior.
package sync

import (
	"sync"
	"testing"
)

// MockLockable is a minimal interface for testing lock operations.
type MockLockable struct {
	mu       sync.Mutex
	locked   bool
	lockCnt  int
	unlockCnt int
}

func (m *MockLockable) Lock() {
	m.mu.Lock()
	m.locked = true
	m.lockCnt++
}

func (m *MockLockable) Unlock() {
	m.locked = false
	m.unlockCnt++
	m.mu.Unlock()
}

func (m *MockLockable) IsLocked() bool {
	return m.locked
}

// ===============================
// RowLocker Tests
// ===============================

// TestRowLockerNew tests that a new RowLocker is properly created.
func TestRowLockerNew(t *testing.T) {
	mock := &MockLockable{}
	rl := NewRowLocker(mock)
	if rl == nil {
		t.Fatal("NewRowLocker returned nil")
	}
	if rl.IsLocked() {
		t.Error("new RowLocker should not be locked")
	}
}

// TestRowLockerLockUnlock tests basic lock and unlock operations.
func TestRowLockerLockUnlock(t *testing.T) {
	mock := &MockLockable{}
	rl := NewRowLocker(mock)

	rl.Lock()
	if !rl.IsLocked() {
		t.Error("RowLocker should be locked after Lock()")
	}
	if mock.lockCnt != 1 {
		t.Errorf("underlying Lock called %d times; want 1", mock.lockCnt)
	}

	rl.Unlock()
	if rl.IsLocked() {
		t.Error("RowLocker should not be locked after Unlock()")
	}
	if mock.unlockCnt != 1 {
		t.Errorf("underlying Unlock called %d times; want 1", mock.unlockCnt)
	}
}

// TestRowLockerDeferSafe tests that RowLocker works with defer.
func TestRowLockerDeferSafe(t *testing.T) {
	mock := &MockLockable{}
	rl := NewRowLocker(mock)

	func() {
		rl.Lock()
		defer rl.Unlock()

		if !rl.IsLocked() {
			t.Error("should be locked within function")
		}
	}()

	if rl.IsLocked() {
		t.Error("should be unlocked after function returns")
	}
	if mock.lockCnt != 1 || mock.unlockCnt != 1 {
		t.Errorf("lock/unlock count mismatch: lock=%d unlock=%d", mock.lockCnt, mock.unlockCnt)
	}
}

// TestRowLockerWithFunc tests the WithLock helper for scoped locking.
func TestRowLockerWithFunc(t *testing.T) {
	mock := &MockLockable{}
	rl := NewRowLocker(mock)

	executed := false
	rl.WithLock(func() {
		executed = true
		if !rl.IsLocked() {
			t.Error("should be locked during callback")
		}
	})

	if !executed {
		t.Error("callback should have been executed")
	}
	if rl.IsLocked() {
		t.Error("should be unlocked after WithLock returns")
	}
}

// TestRowLockerWithFuncPanicSafe tests that WithLock unlocks even on panic.
func TestRowLockerWithFuncPanicSafe(t *testing.T) {
	mock := &MockLockable{}
	rl := NewRowLocker(mock)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
		if rl.IsLocked() {
			t.Error("should be unlocked even after panic")
		}
	}()

	rl.WithLock(func() {
		panic("test panic")
	})
}

// ===============================
// WindowLocker Tests
// ===============================

// MockRefCounted provides a mock for window-like objects with reference counting.
type MockRefCounted struct {
	MockLockable
	ref      int
	owner    int
	closeCnt int
}

func (m *MockRefCounted) Inc() {
	m.ref++
}

func (m *MockRefCounted) Dec() int {
	m.ref--
	return m.ref
}

func (m *MockRefCounted) Ref() int {
	return m.ref
}

func (m *MockRefCounted) SetOwner(owner int) {
	m.owner = owner
}

func (m *MockRefCounted) Owner() int {
	return m.owner
}

func (m *MockRefCounted) Close() {
	m.closeCnt++
}

// TestWindowLockerNew tests that a new WindowLocker is properly created.
func TestWindowLockerNew(t *testing.T) {
	mock := &MockRefCounted{}
	wl := NewWindowLocker(mock)
	if wl == nil {
		t.Fatal("NewWindowLocker returned nil")
	}
	if wl.IsLocked() {
		t.Error("new WindowLocker should not be locked")
	}
}

// TestWindowLockerLockIncreasesRef tests that Lock increments reference count.
func TestWindowLockerLockIncreasesRef(t *testing.T) {
	mock := &MockRefCounted{ref: 1}
	wl := NewWindowLocker(mock)

	wl.Lock('K') // 'K' for keyboard owner
	if mock.ref != 2 {
		t.Errorf("ref count after Lock: got %d; want 2", mock.ref)
	}
	if mock.owner != 'K' {
		t.Errorf("owner after Lock: got %c; want K", mock.owner)
	}
}

// TestWindowLockerUnlockDecreasesRef tests that Unlock decrements reference count.
func TestWindowLockerUnlockDecreasesRef(t *testing.T) {
	mock := &MockRefCounted{ref: 1}
	wl := NewWindowLocker(mock)

	wl.Lock('E')
	wl.Unlock()

	if mock.ref != 1 {
		t.Errorf("ref count after Unlock: got %d; want 1", mock.ref)
	}
	if mock.owner != 0 {
		t.Errorf("owner after Unlock: got %d; want 0", mock.owner)
	}
}

// TestWindowLockerUnlockCallsCloseOnZeroRef tests that Close is called when ref hits zero.
func TestWindowLockerUnlockCallsCloseOnZeroRef(t *testing.T) {
	mock := &MockRefCounted{ref: 0} // Start at 0, Lock will increment to 1
	wl := NewWindowLocker(mock)

	wl.Lock('M')
	if mock.ref != 1 {
		t.Errorf("ref count after Lock: got %d; want 1", mock.ref)
	}

	wl.Unlock()
	// Dec from 1 to 0 should trigger Close
	if mock.closeCnt != 1 {
		t.Errorf("Close called %d times; want 1", mock.closeCnt)
	}
}

// TestWindowLockerDeferSafe tests that WindowLocker works with defer.
func TestWindowLockerDeferSafe(t *testing.T) {
	mock := &MockRefCounted{ref: 1}
	wl := NewWindowLocker(mock)

	func() {
		wl.Lock('E')
		defer wl.Unlock()

		if !wl.IsLocked() {
			t.Error("should be locked within function")
		}
		if mock.ref != 2 {
			t.Errorf("ref during lock: got %d; want 2", mock.ref)
		}
	}()

	if wl.IsLocked() {
		t.Error("should be unlocked after function returns")
	}
	if mock.ref != 1 {
		t.Errorf("ref after unlock: got %d; want 1", mock.ref)
	}
}

// TestWindowLockerWithFunc tests the WithLock helper for scoped locking.
func TestWindowLockerWithFunc(t *testing.T) {
	mock := &MockRefCounted{ref: 1}
	wl := NewWindowLocker(mock)

	executed := false
	wl.WithLock('K', func() {
		executed = true
		if !wl.IsLocked() {
			t.Error("should be locked during callback")
		}
		if mock.owner != 'K' {
			t.Errorf("owner during callback: got %c; want K", mock.owner)
		}
	})

	if !executed {
		t.Error("callback should have been executed")
	}
	if wl.IsLocked() {
		t.Error("should be unlocked after WithLock returns")
	}
}

// TestWindowLockerOwnerTracking tests that owner is properly tracked.
func TestWindowLockerOwnerTracking(t *testing.T) {
	testCases := []struct {
		name  string
		owner int
	}{
		{"Keyboard", 'K'},
		{"Mouse", 'M'},
		{"Error", 'E'},
		{"Command X", 'X'},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockRefCounted{ref: 1}
			wl := NewWindowLocker(mock)

			wl.Lock(tc.owner)
			if mock.owner != tc.owner {
				t.Errorf("owner: got %c; want %c", mock.owner, tc.owner)
			}

			wl.Unlock()
			if mock.owner != 0 {
				t.Errorf("owner after unlock: got %d; want 0", mock.owner)
			}
		})
	}
}

// ===============================
// Lock Ordering Tests
// ===============================

// TestLockOrderingRowThenWindow tests the documented lock ordering: Row -> Window.
func TestLockOrderingRowThenWindow(t *testing.T) {
	rowMock := &MockLockable{}
	winMock := &MockRefCounted{ref: 1}

	rl := NewRowLocker(rowMock)
	wl := NewWindowLocker(winMock)

	// Correct ordering: row first, then window
	rl.Lock()
	wl.Lock('E')

	if !rl.IsLocked() {
		t.Error("row should be locked")
	}
	if !wl.IsLocked() {
		t.Error("window should be locked")
	}

	// Unlock in reverse order
	wl.Unlock()
	rl.Unlock()

	if rl.IsLocked() {
		t.Error("row should be unlocked")
	}
	if wl.IsLocked() {
		t.Error("window should be unlocked")
	}
}

// TestNestedWithLock tests nested WithLock calls follow proper ordering.
func TestNestedWithLock(t *testing.T) {
	rowMock := &MockLockable{}
	winMock := &MockRefCounted{ref: 1}

	rl := NewRowLocker(rowMock)
	wl := NewWindowLocker(winMock)

	lockOrder := []string{}

	rl.WithLock(func() {
		lockOrder = append(lockOrder, "row_locked")
		wl.WithLock('E', func() {
			lockOrder = append(lockOrder, "window_locked")
		})
		lockOrder = append(lockOrder, "window_unlocked")
	})
	lockOrder = append(lockOrder, "row_unlocked")

	expected := []string{"row_locked", "window_locked", "window_unlocked", "row_unlocked"}
	if len(lockOrder) != len(expected) {
		t.Errorf("lock order length: got %d; want %d", len(lockOrder), len(expected))
	}
	for i, v := range expected {
		if i >= len(lockOrder) || lockOrder[i] != v {
			t.Errorf("lock order[%d]: got %q; want %q", i, lockOrder[i], v)
		}
	}
}

// ===============================
// Concurrent Access Tests
// ===============================

// TestRowLockerConcurrentAccess tests that RowLocker properly serializes access.
func TestRowLockerConcurrentAccess(t *testing.T) {
	mock := &MockLockable{}
	rl := NewRowLocker(mock)

	const iterations = 100
	counter := 0
	done := make(chan bool, 2)

	increment := func() {
		for i := 0; i < iterations; i++ {
			rl.WithLock(func() {
				counter++
			})
		}
		done <- true
	}

	go increment()
	go increment()

	<-done
	<-done

	if counter != iterations*2 {
		t.Errorf("counter: got %d; want %d (race condition detected)", counter, iterations*2)
	}
}

// TestWindowLockerConcurrentAccess tests that WindowLocker properly serializes access.
func TestWindowLockerConcurrentAccess(t *testing.T) {
	mock := &MockRefCounted{ref: 1}
	wl := NewWindowLocker(mock)

	const iterations = 100
	counter := 0
	done := make(chan bool, 2)

	increment := func() {
		for i := 0; i < iterations; i++ {
			wl.WithLock('K', func() {
				counter++
			})
		}
		done <- true
	}

	go increment()
	go increment()

	<-done
	<-done

	if counter != iterations*2 {
		t.Errorf("counter: got %d; want %d (race condition detected)", counter, iterations*2)
	}
}
