# Rich Text Implementation Plan

## Status Legend
- `[ ]` = not done
- `[x]` = done

## Phase 1: Package Scaffold

### 1.1 Create rich/ Package Structure
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Create rich/frame_test.go with TestNewFrame placeholder |
| Code written | [x] | Create rich/ directory with style.go, span.go, box.go, frame.go stubs |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 97d6f29 |

### 1.2 Style Type
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDefaultStyle, TestStyleEquality |
| Code written | [x] | Style struct with Fg, Bg, Bold, Italic, Scale fields |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit bcb4af7 |

### 1.3 Span Type
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestSpanLen, TestContentLen, TestPlainContent |
| Code written | [x] | Span struct, Content type, Plain() and Len() methods |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit b530bde |

### 1.4 Box Type
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestBoxIsNewline, TestBoxIsTab |
| Code written | [x] | Box struct with Text, Nrune, Bc, Style, Wid fields |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 94d26c2 |

## Phase 2: Minimal Frame Rendering

### 2.1 Frame Interface Definition
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - interface only |
| Code written | [x] | Frame interface with Init, SetContent, Rect, Redraw methods |
| Tests pass | [x] | Compiles |
| Code committed | [x] | Commit 97d6f29 (part of scaffold) |

### 2.2 Frame Implementation Scaffold
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestFrameInit with mock display |
| Code written | [x] | frameImpl struct, Init() stores rect and display |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 5b445fc |

### 2.3 Background Color Rendering
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestFrameRedrawFillsBackground |
| Code written | [x] | Redraw() fills rect with distinct background color |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 2dd6627 |

### 2.4 Visual Demo in Edwood
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - visual demo |
| Code written | [x] | Add temporary test hook to create rich.Frame in Edwood, visible different color |
| Tests pass | [x] | Manual verification - see colored rectangle |
| Code committed | [x] | Commit 4cb99c5 |

## Phase 3: Text Layout

### 3.1 Font Loading
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestFrameWithFont |
| Code written | [x] | Frame stores font reference, Option for setting font |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 98dbc01 |

### 3.2 Content to Boxes Conversion
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestContentToBoxes for plain text, newlines, tabs |
| Code written | [x] | contentToBoxes() splits spans into boxes |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit e8a1ac4 |

### 3.3 Box Width Calculation
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestBoxWidth for text, tabs |
| Code written | [x] | boxWidth() using font metrics |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 80234de |

### 3.4 Line Layout Algorithm
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLayoutSingleLine, TestLayoutWrapping, TestLayoutBoxPositions |
| Code written | [x] | layout() positions boxes into lines, handles wrapping |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit cd3b395 |

### 3.5 Draw Plain Text
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawText verifies bytes rendered |
| Code written | [x] | drawText() renders boxes using display.Bytes() |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit b7be488 |

### 3.6 Visual Demo - Plain Text
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - visual demo |
| Code written | [x] | Update demo to show plain text in rich frame |
| Tests pass | [x] | Manual verification - see text rendered |
| Code committed | [x] | Commit e7139ef |

## Phase 4: Styled Text Rendering

### 4.1 Color Support
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawTextWithColor, TestDrawTextWithMultipleColors, TestDrawTextWithDefaultColor |
| Code written | [x] | drawText() uses box style colors |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit c0089ed |

### 4.2 Font Variant Support
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestFontVariantsBoldText, TestFontVariantsItalicText, TestFontVariantsBoldItalicText, TestFontVariantsFallbackToRegular, TestFontVariantsMixedContent |
| Code written | [x] | fontForStyle() selects font based on Bold/Italic, drawText() uses per-box font |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 3b500fd |

### 4.3 Font Scale Support
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestFontScaleH1Text, TestFontScaleH2Text, TestFontScaleH3Text, TestFontScaleFallbackToRegular, TestFontScaleMixedContent, TestFontScaleWithBoldCombination |
| Code written | [x] | Scale-aware font selection or rendering |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit e8e7948 |

### 4.3a Export Font Variant Options
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Existing tests in frame_test.go use these options |
| Code written | [x] | Move WithBoldFont, WithItalicFont, WithBoldItalicFont, WithScaledFont from frame_test.go to rich/options.go |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 5f810e4 |

### 4.3b Wire Font Variants in Demo
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - visual demo |
| Code written | [x] | Update DemoFrame() to load bold/italic font variants and pass via exported options |
| Tests pass | [x] | Manual verification - bold/italic text renders differently |
| Code committed | [x] | |

### 4.4 Visual Demo - Styled Text
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - visual demo |
| Code written | [x] | Demo with multiple styles: heading, bold, italic, colors |
| Tests pass | [x] | Manual verification - see styled text (note: spacing issue between bold/regular) |
| Code committed | [x] | |

## Phase 5: Coordinate Mapping

### 5.1 Ptofchar Implementation
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPtofchar for start, middle, end, wrapped lines |
| Code written | [x] | Ptofchar() maps rune offset to screen point |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 600a0c8 |

### 5.2 Charofpt Implementation
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestCharofpt for various screen positions |
| Code written | [x] | Charofpt() maps screen point to rune offset |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 320d176 |

### 5.3 Coordinate Round-Trip
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestCoordinateRoundTrip verifies Charofpt(Ptofchar(n)) == n |
| Code written | [x] | Ensure consistency between mappings |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 6bf0dc3 |

## Phase 6: Selection

### 6.1 Selection State
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestSetSelection, TestGetSelection |
| Code written | [x] | Frame stores p0, p1 selection bounds |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 7c27e13 |

### 6.2 Selection Drawing
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawSelection highlights correct region |
| Code written | [x] | drawSelection() renders highlight background |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 5540491 |

### 6.3 Mouse Selection
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestSelectWithMouse (mock mousectl) |
| Code written | [x] | Select() handles mouse drag, returns p0, p1 |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit f9f2866 |

### 6.4 Visual Demo - Selection
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - visual demo |
| Code written | [x] | Demo with clickable, selectable text |
| Tests pass | [x] | Manual verification - can select text |
| Code committed | [x] | Commit a2b6c00 |

## Phase 7: Scrolling

### 7.1 Origin Tracking
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestSetOrigin, TestGetOrigin, TestOriginZero, TestOriginClear, TestOriginUpdateOverwrites |
| Code written | [x] | Frame stores origin offset |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit fcb1f25 |

### 7.2 Partial Content Display
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDisplayFromOrigin |
| Code written | [x] | Redraw starts from origin, not beginning |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 72310f9 |

### 7.3 Frame Fill Status
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestMaxLines*, TestVisibleLines*, TestFull* in scroll_test.go |
| Code written | [x] | Full(), MaxLines(), VisibleLines() methods |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 54a9c9a |

### 7.4 Visual Demo - Scrolling
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - visual demo |
| Code written | [x] | Demo with scrollable content |
| Tests pass | [x] | Manual verification - can scroll |
| Code committed | [x] | Commit 64f093b |

## Phase 8: RichText Component

### 8.1 RichText Struct
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestRichTextInit, TestRichTextScrollRect, TestRichTextFrameRect, TestRichTextSetContent, TestRichTextSelection, TestRichTextOrigin, TestRichTextRedraw |
| Code written | [x] | RichText struct wrapping rich.Frame |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 6141b9b |

### 8.2 Scrollbar Rendering
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestScrollbarPosition, TestScrollbarThumbAtTop, TestScrollbarThumbAtBottom, TestScrollbarThumbMiddle, TestScrollbarNoContent, TestScrollbarContentFits |
| Code written | [x] | scrDraw() renders scrollbar thumb |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 6541d1d |

### 8.3 Scrollbar Interaction
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestScrollbarClickButton1, TestScrollbarClickButton2, TestScrollbarClickButton3, TestScrollbarClickAtTop, TestScrollbarClickAtBottom, TestScrollbarClickNoContent, TestScrollbarClickContentFits |
| Code written | [x] | Handle scrollbar mouse events |
| Tests pass | [x] | |
| Code committed | [x] | Commit 6b85ee2 |

### 8.4 Mouse Wheel Scrolling
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [ ] | TestMouseWheelScroll |
| Code written | [ ] | Handle scroll wheel events |
| Tests pass | [ ] | |
| Code committed | [ ] | |

## Phase 9: Markdown Parser

### 9.1 Create markdown/ Package
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [ ] | TestParsePlainText |
| Code written | [ ] | markdown/ directory with parse.go |
| Tests pass | [ ] | |
| Code committed | [ ] | |

### 9.2 Parse Headings
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [ ] | TestParseH1 through TestParseH6 |
| Code written | [ ] | Detect # prefix, apply heading styles |
| Tests pass | [ ] | |
| Code committed | [ ] | |

### 9.3 Parse Bold/Italic
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [ ] | TestParseBold, TestParseItalic, TestParseBoldItalic |
| Code written | [ ] | Detect **bold**, *italic*, ***both*** |
| Tests pass | [ ] | |
| Code committed | [ ] | |

### 9.4 Parse Code Spans
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [ ] | TestParseInlineCode |
| Code written | [ ] | Detect `code` spans |
| Tests pass | [ ] | |
| Code committed | [ ] | |

## Phase 10: Preview Integration

### 10.1 Preview Window Type
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [ ] | TestPreviewWindowCreation |
| Code written | [ ] | PreviewWindow struct or Window rich mode |
| Tests pass | [ ] | |
| Code committed | [ ] | |

### 10.2 Preview Command
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [ ] | N/A - integration |
| Code written | [ ] | "Preview" tag command opens preview |
| Tests pass | [ ] | Manual verification |
| Code committed | [ ] | |

### 10.3 Live Update
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [ ] | TestPreviewUpdatesOnChange |
| Code written | [ ] | Preview re-renders when source changes |
| Tests pass | [ ] | |
| Code committed | [ ] | |

### 10.4 Scroll Sync (Optional)
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [ ] | TestScrollSync |
| Code written | [ ] | Preview scrolls with source |
| Tests pass | [ ] | |
| Code committed | [ ] | |

---

## Current Task

**Next unfinished task**: 1.1 Create rich/ Package Structure - Tests exist

## Test Summary

| Suite | Count | File Location |
|-------|-------|---------------|
| Style | 2 | rich/style_test.go |
| Span | 3 | rich/span_test.go |
| Box | 2 | rich/box_test.go |
| Frame Init | 2 | rich/frame_test.go |
| Layout | 4 | rich/layout_test.go |
| Coordinates | 4 | rich/coords_test.go |
| Selection | 3 | rich/select_test.go |
| Scrolling | 3 | rich/scroll_test.go |
| Markdown | 8 | markdown/parse_test.go |
| Integration | 4 | richtext_test.go |
| **Total** | **~35** | |

## How to Run Tests

```bash
# All rich text tests
go test ./rich/... ./markdown/...

# With verbose output
go test -v ./rich/...

# Specific package
go test ./rich/
```

## Files

| File | Purpose |
|------|---------|
| docs/richtext-design.md | Design document and architecture |
| PLAN.md | This file - implementation tracking |
| rich/style.go | Style type definition |
| rich/span.go | Span and Content types |
| rich/box.go | Box type for layout |
| rich/frame.go | RichFrame implementation |
| rich/select.go | Selection handling |
| rich/options.go | Functional options |
| markdown/parse.go | Markdown to Content parser |
| richtext.go | RichText component |
| preview.go | Preview window integration |

---

## Known Issues

### Selection Not Displayed During Drag
| Stage | Status | Notes |
|-------|--------|-------|
| Issue identified | [x] | Hand-validation: text is selectable but selection highlight doesn't update as you drag |
| Root cause found | [ ] | Select() loop likely not calling Redraw() during drag |
| Fix implemented | [ ] | |
| Fix tested | [ ] | |
| Fix committed | [ ] | |
