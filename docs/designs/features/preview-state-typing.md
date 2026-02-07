# PreviewState Typing Design

## Problem

`wind/preview.go` uses `interface{}` stubs for three fields:

```go
sourceMap  interface{} // *markdown.SourceMap
linkMap    interface{} // *markdown.LinkMap
imageCache interface{} // *rich.ImageCache
```

These lose type safety, require type assertions at every use site, and prevent the compiler from catching misuse. The goal is to replace them with proper types.

## Dependency Analysis

Current dependency graph (relevant packages):

```
main package (wind.go, exec.go, etc.)
    ├── markdown/     (SourceMap, LinkMap, ParseWithSourceMap)
    ├── rich/         (ImageCache, Frame, Content, Style)
    └── wind/         (PreviewState, WindowBase, etc.)

markdown/
    └── rich/         (Style, Span, Content types only)

rich/
    └── (no project deps besides edwood/draw)

wind/
    └── (no project deps — only stdlib "image")
```

Key observation: **No package currently imports `wind/`**. The `wind/` package is being built bottom-up as a future home for Window-related types. The main package does not yet embed `WindBase` or use `wind.PreviewState` for its actual preview operations — it has its own `previewSourceMap *markdown.SourceMap` etc. directly on `Window` in `wind.go`.

## Options Evaluated

### Option A: wind/ imports markdown/ and rich/ directly

Replace `interface{}` with concrete types:

```go
import (
    "github.com/rjkroege/edwood/markdown"
    "github.com/rjkroege/edwood/rich"
)

type PreviewState struct {
    sourceMap  *markdown.SourceMap
    linkMap    *markdown.LinkMap
    imageCache *rich.ImageCache
    // ...
}
```

**Pros:**
- Simplest change — direct types, no interfaces, no indirection.
- Matches exactly how the main package (`wind.go:74-77`) already uses these types.
- Compiler enforces correct types at all call sites.
- No new packages or abstractions needed.

**Cons:**
- Adds `markdown/` and `rich/` as dependencies of `wind/`.
- If `markdown/` or `rich/` ever needed to import `wind/`, it would create a cycle.

**Circular dependency risk assessment:**
- `markdown/` imports `rich/` (for `Style`, `Span`, `Content`). It has no reason to import `wind/` — it is a pure parser.
- `rich/` imports `edwood/draw`. It has no reason to import `wind/` — it is a rendering engine.
- The dependency direction is clear: `wind/` is a consumer of `markdown/` and `rich/` outputs, not the other way around. This is the same direction the main package already follows.
- The design doc's architecture diagram confirms this: `wind/` sits above `markdown/` and `rich/` in the dependency tree.

**Verdict: No realistic circular dependency risk.**

### Option B: Define interfaces in wind/

```go
// In wind/preview.go
type SourceMapper interface {
    ToSource(renderedStart, renderedEnd int) (srcStart, srcEnd int)
    ToRendered(srcRuneStart, srcRuneEnd int) (renderedStart, renderedEnd int)
}

type LinkMapper interface {
    URLAt(pos int) string
}

type ImageCacher interface {
    Get(path string) (interface{}, bool)
    Load(path string) (interface{}, error)
    Clear()
}
```

**Pros:**
- `wind/` stays dependency-free.
- Allows alternative implementations for testing.

**Cons:**
- Adds abstraction for no practical benefit — there is exactly one implementation of each.
- `ImageCacher` is awkward: `Get()` and `Load()` return `*rich.CachedImage`, which would need to become `interface{}` or a new interface, losing type safety on the return value.
- The main package already uses concrete types (`*markdown.SourceMap`, etc.) — wrapping in interfaces adds friction at the integration boundary.
- Interfaces in Go should be defined by consumers, but `wind/` doesn't actually *use* these types — it stores and returns them. The actual consumers are in the main package.

**Verdict: Over-engineered for this case.**

### Option C: Shared types package

Create a `types/` or `preview/` package that defines the types both `wind/` and `markdown/`/`rich/` would use.

**Pros:**
- Breaks any potential cycle.

**Cons:**
- `SourceMap`, `LinkMap`, and `ImageCache` are not just type definitions — they have complex implementations with methods, internal state, and dependencies (HTTP client for image loading, rune mapping tables for source maps). Moving them to a shared package would mean either:
  - Moving the implementations too (destroying the `markdown/` and `rich/` package organization), or
  - Defining interfaces (same problems as Option B).
- Adds package proliferation for no benefit.

**Verdict: Wrong tool for this problem.**

## Decision: Option A

Import `markdown/` and `rich/` directly into `wind/`. This is the simplest correct solution, matches the existing code patterns in the main package, and has no realistic circular dependency risk.

## Implementation Plan

### Changes to `wind/preview.go`

1. Add imports for `markdown` and `rich` packages.
2. Replace `interface{}` fields with concrete types:
   - `sourceMap interface{}` → `sourceMap *markdown.SourceMap`
   - `linkMap interface{}` → `linkMap *markdown.LinkMap`
   - `imageCache interface{}` → `imageCache *rich.ImageCache`
3. Update getter/setter signatures:
   - `SourceMap() interface{}` → `SourceMap() *markdown.SourceMap`
   - `SetSourceMap(interface{})` → `SetSourceMap(*markdown.SourceMap)`
   - `LinkMap() interface{}` → `LinkMap() *markdown.LinkMap`
   - `SetLinkMap(interface{})` → `SetLinkMap(*markdown.LinkMap)`
   - `ImageCache() interface{}` → `ImageCache() *rich.ImageCache`
   - `SetImageCache(interface{})` → `SetImageCache(*rich.ImageCache)`

### Changes to `wind/window_test.go`

Update existing tests to use typed values instead of `interface{}`:
- `TestPreviewStateSourceMap`: Create a real `*markdown.SourceMap` (or verify nil returns typed nil).
- `TestPreviewStateLinkMap`: Create a real `*markdown.LinkMap`.
- `TestPreviewStateImageCache`: Create a real `*rich.ImageCache`.
- `TestPreviewStateClearCache`: Set typed values, verify clearing works.

### No changes needed

- `wind/window.go`: `WindowBase` delegates to `PreviewState` — no `interface{}` in its API that relates to these fields.
- Main package (`wind.go`): Already uses concrete types on its own `Window` struct. The `wind.PreviewState` is not yet integrated into the main `Window` — that integration is a separate future task.
- `markdown/`, `rich/`: No changes.

## Test Strategy

The existing tests in `wind/window_test.go` already test PreviewState getters/setters with nil checks. After the type change:

1. Nil checks (`ps.SourceMap() != nil`) continue to work identically for typed nil pointers.
2. Add tests that set real typed values (e.g., `markdown.NewLinkMap()`, `rich.NewImageCache(10)`) and verify round-trip through getter/setter.
3. Verify `ClearCache()` sets all three fields to typed nil.
4. `ParseWithSourceMap()` can be called in tests to produce a real `*SourceMap` for testing.
