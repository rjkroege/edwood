// Package wind provides the Window type and related components for edwood.
//
// This package is a scaffold for extracting window-related functionality
// from the main package. The full Window type will be moved here in later
// phases of the refactoring plan.
//
// Current components:
//   - WindowState: file descriptor tracking, addresses, dirty flags
//   - PreviewState: preview mode fields for rich text rendering
//
// Future components (to be added in later phases):
//   - Window: the main window struct (Phase 5F)
//   - Drawing methods (Phase 5D)
//   - Event handling (Phase 5E)
package wind

// This file will contain the Window type when it is moved from the main package.
// For now, it serves as documentation and a placeholder for the package.
//
// The Window type manages:
//   - Display and drawing context
//   - Tag and body text areas
//   - Window state (dirty, addresses, limits)
//   - Preview mode state
//   - Event handling
//   - File server integration (nopen, ctlfid)
