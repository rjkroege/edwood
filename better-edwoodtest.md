# Extending edwoodtest with Pixel Rendering

## Goal

Capture before/after raster images of frame rendering so that tests can assert
visual correctness without depending on implementation-internal draw operation
strings. SVG step-by-step traces are useful for debugging individual ops but
produce nothing a human (or test) can use to verify "the screen looks right
after this delete." PNG golden files fill that gap and are more stable than
drawops strings across refactors that preserve visual output.

---

## Current state

`edwoodtest` has three types:

| Type | Role | Pixel backing? |
|---|---|---|
| `mockDisplay` | Implements `draw.Display` | No |
| `mockImage` | Implements `draw.Image` | No — only records strings |
| `mockFont` | Implements `draw.Font` | No — fixed-width stub |

`mockImage.Draw` appends a string such as
`"blit (20,30)-(60,40) …, to (20,20)-(60,30) …"` and also accumulates an SVG
fragment, but it never writes to any pixel buffer. The `screenimage` field of
`mockDisplay` is therefore a completely opaque object; there is no way to
produce a PNG of the current screen state.

`mockFont` carries only `(width int, height int)` and fakes `BytesWidth` as
`width × runeCount`. This is intentional for the existing drawops tests because
it lets coordinates be expressed in neat character multiples. It must not be
broken by the new work.

`draw.Display.ScaleSize` returns the constant `1` regardless of DPI, which
means HiDPI behavior is never exercised.

---

## Required dependencies

Both dependencies are already in `go.mod` as indirect transitive deps of
`duitdraw`:

```
github.com/golang/freetype   (TrueType rasterizer)
golang.org/x/image           (font.Drawer, plan9font, gofont/goregular)
```

No new `go get` is needed.

---

## Proposed changes

### 1. `mockDisplay` — add DPI and a pixel-backed screen image

**New field:**
```go
type mockDisplay struct {
    // existing fields …
    dpi         int         // 0 means "not a rendering display"
    pixscreen   *image.RGBA // non-nil only when dpi > 0
}
```

**`ScaleSize` fix:**
```go
func (d *mockDisplay) ScaleSize(n int) int {
    if d.dpi <= 100 {
        return n
    }
    return (n*d.dpi + 50) / 100
}
```
Mirrors duitdraw's implementation exactly.

**New constructor:**
```go
// NewDisplayWithDPI returns a mock display that renders to an actual *image.RGBA.
// Calling ScreenImageAsPNG on the returned GettableDrawOps produces a real PNG.
// dpi controls ScaleSize; 100 is "1:1 logical-to-physical".
func NewDisplayWithDPI(rectofi image.Rectangle, dpi int) draw.Display
```

`NewDisplayWithDPI` initializes `pixscreen` as:
```go
d.pixscreen = image.NewRGBA(image.Rect(0, 0, 800, 600))
draw.Draw(d.pixscreen, d.pixscreen.Bounds(), image.White, image.Point{}, draw.Src)
```
and wraps it in a `mockImage` whose `m` field points to `d.pixscreen`.

The existing `NewDisplay` is unchanged and leaves `pixscreen == nil`, so all
existing tests keep working.

**Pre-allocated solid colours** (`White`, `Black`, `Opaque`, `Transparent`)
return images whose `m` field is `image.NewUniform(c)` — uniform colour images,
cheap and exactly what duitdraw does. `AllocImage` for a 1×1 replicated colour
does the same. Larger `AllocImage` calls return an `*image.RGBA` pre-filled
with the colour, exactly as duitdraw does.

### 2. `mockImage` — add a pixel backing field

**New field:**
```go
type mockImage struct {
    // existing fields …
    m image.Image // nil for the non-rendering path; *image.RGBA or *image.Uniform otherwise
}
```

**`Draw` update:**

After the existing string-recording logic, add a rendering branch:
```go
func (i *mockImage) Draw(r image.Rectangle, src, mask draw.Image, p1 image.Point) {
    // … existing string recording and SVG accumulation unchanged …

    // Pixel-rendering path: only when this image has an RGBA backing.
    dst, ok := i.m.(*image.RGBA)
    if !ok {
        return
    }
    msrc, ok := src.(*mockImage)
    if !ok || msrc.m == nil {
        return
    }
    if mask == nil {
        stdDraw.Draw(dst, r, msrc.m, p1, stdDraw.Src)
    } else {
        mmask := mask.(*mockImage)
        stdDraw.DrawMask(dst, r, msrc.m, p1, mmask.m, p1, stdDraw.Over)
    }
}
```

`stdDraw` is `"image/draw"` aliased to avoid collision with the `draw` package.

**`Bytes` update:**

Two rendering strategies are described in §3 below. In both cases, the
advance-width logic (`f.BytesWidth(b)`) is unchanged so existing coordinates
stay correct.

**`Border` update:**

Use `golang.org/x/exp/shiny/imageutil.Border` (already pulled in by duitdraw's
transitive deps) to fill the border rectangles into the RGBA image.

### 3. Text rendering options for `mockImage.Bytes`

Two strategies, ordered by stability preference:

#### Option A — Filled rectangle (recommended starting point)

`Bytes` fills the bounding box of the string with the source colour instead of
drawing real glyphs:
```go
if dst, ok := i.m.(*image.RGBA); ok {
    box := image.Rectangle{
        Min: pt,
        Max: pt.Add(image.Pt(f.BytesWidth(b), f.Height())),
    }
    stdDraw.Draw(dst, box, msrc.m, image.Point{}, stdDraw.Src)
}
```

Pros: platform-independent, deterministic, no real font needed, golden PNGs
never change due to font hinting or sub-pixel differences.

Cons: PNGs show coloured rectangles rather than readable text; suitable for
verifying *layout geometry* (positions and sizes of text runs, fills, blits)
but not glyph appearance.

#### Option B — Real glyphs (future work)

Wrap a `font.Face` inside `mockFont`:
```go
type mockFont struct {
    width, height int
    face          font.Face // nil on the non-rendering path
}

func NewFontFromFace(face font.Face) draw.Font {
    m := face.Metrics()
    return &mockFont{
        face:   face,
        height: (m.Ascent + m.Descent).Round(),
    }
}
```

When `face != nil`, `Bytes` uses `font.Drawer`:
```go
if dst, ok := i.m.(*image.RGBA); ok && mf.face != nil {
    ascent := mf.face.Metrics().Ascent
    dot := fixed.P(pt.X, pt.Y).Add(fixed.Point26_6{Y: ascent})
    (&font.Drawer{Dst: dst, Src: msrc.m, Face: mf.face, Dot: dot}).DrawBytes(b)
}
```

And `BytesWidth`/`RunesWidth`/`StringWidth` query glyph advances from the face.

Use GoRegular (built in, no file I/O) or a Plan 9 bitmap font loaded from the
PLAN9 tree for determinism.

The risk is that sub-pixel glyph positioning differs across Go versions or
platforms, making PNG golden files brittle. Start with Option A; add Option B
when it's clear the golden files are stable.

### 4. `GettableDrawOps` — new method

```go
type GettableDrawOps interface {
    DrawOps() []string
    Clear()
    SVGDrawOps(w io.Writer) error
    ScreenImageAsPNG(w io.Writer) error // NEW
}
```

Implementation on `mockDisplay`:
```go
func (d *mockDisplay) ScreenImageAsPNG(w io.Writer) error {
    if d.pixscreen == nil {
        return errors.New("ScreenImageAsPNG: display not created with NewDisplayWithDPI")
    }
    return png.Encode(w, d.pixscreen)
}
```

### 5. Changes to `mockDisplay.AllocImageMix`

Currently `AllocImageMix` blends two colours arithmetically and returns a
1×1 replicated `mockImage`. For the rendering path the same logic applies but
the colour must be stored as an `image.Uniform` so that `Draw` can use it as a
source:
```go
func (d *mockDisplay) AllocImageMix(color1, color3 draw.Color) draw.Image {
    c1 := draw.WithAlpha(color1, 0x3f) >> 8
    c3 := draw.WithAlpha(color3, 0xbf) >> 8
    c := ((c1 + c3) << 8) | 0xff

    mi := &mockImage{d: d, r: image.Rect(0, 0, 1, 1), repl: true, c: c}
    if d.pixscreen != nil {
        mi.m = image.NewUniform(colorFromDrawColor(c)) // helper to convert draw.Color → color.RGBA
    }
    return mi
}
```

The same conditional-`m`-setting pattern applies to `AllocImage`.

---

## New test pattern

A new setup helper alongside the existing `setupFrame`:

```go
// setupFrameRendering creates a frame backed by a pixel-rendering display.
// dpi is typically 100 (1:1). The returned GettableDrawOps can produce PNGs.
func setupFrameRendering(t *testing.T, iv *invariants, dpi int) Frame {
    t.Helper()
    display := edwoodtest.NewDisplayWithDPI(iv.textarea, dpi)
    // … same colour allocation as setupFrame …
    return fr
}
```

A test snapshot helper:

```go
func snapPNG(t *testing.T, fr Frame, suffix string) {
    t.Helper()
    path := testName(t, suffix) // e.g. "testdata/TestDelete/deleteEliminatesSoftWrap_before.png"
    f, err := os.Create(path)
    if err != nil {
        t.Fatalf("snapPNG create: %v", err)
    }
    defer f.Close()
    if err := gdo(t, fr).(edwoodtest.GettableDrawOps).ScreenImageAsPNG(f); err != nil {
        t.Fatalf("snapPNG encode: %v", err)
    }
}
```

Usage in a test:

```go
fr := setupFrameRendering(t, iv, 100)
fr.Insert([]rune("0abX\n1cd"), 0)
snapPNG(t, fr, "_before")

fr.Delete(3, 4)
snapPNG(t, fr, "_after")

comparePixelGolden(t, "_before")
comparePixelGolden(t, "_after")
```

`comparePixelGolden` loads `testdata/<test-name>_<suffix>.png` (the baseline),
pixel-diffs it against the just-written file, and fails with an annotation if
they differ. Golden files are regenerated with a `-update` flag, the same
pattern used by the existing SVG baselines.

---

## Non-goals

- Do not break existing `drawops` string tests or SVG baseline tests.
- Do not make real font rendering the default; keep `mockFont` as-is for
  existing callers.
- Do not add DPI-scaled rendering for the existing `NewDisplay` path.

---

## Open questions

1. **`-update` flag placement.** The existing `*validate` flag lives inside the
   `frame` package. A new `-update` flag for PNG goldens could sit alongside it
   or in `edwoodtest`. Prefer `edwoodtest` for reuse across packages.

2. **Pixel-diff tolerance.** Should `comparePixelGolden` be strict (byte-for-byte)
   or allow a small per-pixel delta? Start strict; loosen only if anti-aliasing
   in Option B proves unavoidable.

3. **Screen size.** The hardcoded 800×600 `pixscreen` is sufficient for frame
   tests. If larger tests appear, `NewDisplayWithDPI` should accept explicit
   screen dimensions rather than hardcoding them.

4. **`mockImage.m` for non-screen images.** The screen image needs `*image.RGBA`
   for rendering. Intermediate images (colours, backgrounds) need only
   `image.Uniform`. Make sure `Draw` dispatches correctly on the concrete type
   of `msrc.m` so that `image.Uniform` sources work (they implement
   `image.Image` and the stdlib draw package handles them natively).
