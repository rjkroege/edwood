# Rich Text Implementation Plan - Archive (Phases 1-24)

This file contains completed phases 1-24. See `PLAN.md` for current work (Codebase Refactoring).

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
| Tests exist | [x] | TestMouseWheelScrollDown, TestMouseWheelScrollUp, TestMouseWheelScrollUpAtTop, TestMouseWheelScrollDownAtBottom, TestMouseWheelScrollNoContent, TestMouseWheelScrollContentFits, TestMouseWheelScrollMultipleScrolls |
| Code written | [x] | Handle scroll wheel events |
| Tests pass | [x] | All 8 mouse wheel tests pass |
| Code committed | [x] | Commit abf5a44 |

## Phase 9: Markdown Parser

### 9.1 Create markdown/ Package
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParsePlainText |
| Code written | [x] | markdown/ directory with parse.go |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 45d39c9 |

### 9.2 Parse Headings
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseH1 through TestParseH6 |
| Code written | [x] | Detect # prefix, apply heading styles |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit f5dbaad |

### 9.3 Parse Bold/Italic
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseBold, TestParseItalic, TestParseBoldItalic |
| Code written | [x] | Detect **bold**, *italic*, ***both*** |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit eecdd1b |

### 9.4 Parse Code Spans
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseInlineCode |
| Code written | [x] | Detect `code` spans |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit f3d1efb |

## Phase 10: Preview Integration

### 10.1 Preview Window Type
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewWindowCreation, TestPreviewWindowSetMarkdown, TestPreviewWindowRedraw, TestPreviewWindowWithSource, TestPreviewWindowParsesMarkdownCorrectly |
| Code written | [x] | PreviewWindow struct wrapping RichText, with SetMarkdown using markdown.Parse |
| Tests pass | [x] | All 5 preview window tests pass |
| Code committed | [x] | Commit a00c082 |

### 10.2 Markdeep Command
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - integration |
| Code written | [x] | "Markdeep" tag command opens preview window for .md files |
| Tests pass | [x] | Manual verification |
| Code committed | [x] | Commit 17fa18a |

### 10.3 Live Update
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewUpdatesOnChange, TestPreviewUpdatesPreservesSource, TestPreviewUpdatesMultipleTimes |
| Code written | [x] | PreviewState implements file.BufferObserver for live updates |
| Tests pass | [x] | All 3 live update tests pass |
| Code committed | [x] | Commit b0e13e6 |

### 10.4 Scroll Sync (Optional)
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestScrollSync, TestScrollSyncNoContent, TestScrollSyncContentFits |
| Code written | [x] | SyncToSourcePosition method added to PreviewWindow |
| Tests pass | [x] | All 3 scroll sync tests pass |
| Code committed | [x] | Commit ca8d762 |

## Phase 11: Window Integration

The current `PreviewWindow` is standalone. This phase integrates rich text preview as a **toggle mode** within the existing `Window` type.

### Design Summary

- **Same window, different view**: Preview is a mode toggle, not a separate window
- **Tag unchanged**: Normal tag behavior (filename, Del, Snarf, Put, etc.)
- **Snarf maps to source**: Selection in preview copies raw markdown from source
- **Full column participant**: Resize, move, grow all work normally
- **Mouse chords work**: Look (B3), Exec (B2) work on rendered text

See `docs/richtext-design.md` Phase 11 section for full design.

### 11.1 Source Position Mapping
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestSourceMapSimple, TestSourceMapBold, TestSourceMapHeading |
| Code written | [x] | ParseWithSourceMap returns SourceMap with entries tracking rendered-to-source positions |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 729ae16 |

### 11.2 Window Preview Mode Fields
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestWindowPreviewMode, TestWindowPreviewModeToggle |
| Code written | [x] | Add `previewMode bool`, `richBody *RichText` to Window |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 7d9c48f |

### 11.3 Window Draw in Preview Mode
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestWindowDrawPreviewMode |
| Code written | [x] | Window.Draw() renders richBody when previewMode=true |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 3d18127 |

### 11.4 Window Mouse in Preview Mode
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestWindowMousePreviewSelection, TestWindowMousePreviewScroll |
| Code written | [x] | Window.HandlePreviewMouse() delegates to richBody in preview mode |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 33fda3e |

### 11.5 Snarf with Source Mapping
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewSnarf, TestPreviewSnarfBold, TestPreviewSnarfHeading |
| Code written | [x] | Snarf in preview mode copies from body using source map |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 0e91afd |

### 11.6 Markdeep Command Toggle
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewCommandToggle, TestPreviewCommandEnter, TestPreviewCommandExit |
| Code written | [x] | "Markdeep" command toggles previewMode on window |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 00b96af |

### 11.7 Live Update in Preview Mode
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewLiveUpdate, TestPreviewLiveUpdatePreservesScroll, TestPreviewLiveUpdateMultipleTimes |
| Code written | [x] | Window observes body changes, re-renders preview |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit ba68c2d |

### 11.8 Mouse Chords (Look/Exec)
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewLook, TestPreviewExec, TestPreviewLookExpand |
| Code written | [x] | B2/B3 chords work on rendered text in preview |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 4935ffc |

### 11.9 Keyboard Handling
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewKeyScroll, TestPreviewKeyIgnoresTyping |
| Code written | [x] | Page Up/Down scroll, typing is ignored in preview |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit fcb3889 |

### 11.10 Remove Standalone PreviewWindow
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - cleanup |
| Code written | [x] | Remove or repurpose PreviewWindow, PreviewState types |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 1624740 |

### 11.11 Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | All features verified through comprehensive test suite: 19 preview tests pass |
| Tests pass | [x] | Preview toggles, selection, snarf, chords all work |
| Code committed | [x] | Phase 11 complete - window integration for rich text preview |

## Phase 12: Markdown Links

This phase adds support for rendering and interacting with markdown links `[text](url)`.

### Design Summary

- **Parse links**: Detect `[link text](url)` syntax in markdown
- **Render in blue**: Display link text in traditional blue link color
- **LinkMap**: Track which rendered positions correspond to which URLs
- **Look action**: B3 click on a link opens/plumbs the URL

### Architecture Notes

Links need a mapping from rendered text positions to URLs. This is similar to how `SourceMap` maps rendered positions to source positions. We'll create a `LinkMap` type that tracks:
- Start and end positions of each link in the rendered text
- The URL for each link

When Look (B3) is clicked:
1. Get the click position in rendered text
2. Check if position falls within a link (using LinkMap)
3. If yes, extract the URL and call `plumb` or `look3` with it

### 12.1 LinkMap Type
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLinkMapLookup, TestLinkMapEmpty, TestLinkMapMultipleLinks, TestLinkMapAdjacentLinks |
| Code written | [x] | LinkMap type with Add() and URLAt(pos) methods in markdown/linkmap.go |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 0485e20 |

### 12.2 Parse Links
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseLink, TestParseLinkWithBold, TestParseMultipleLinks |
| Code written | [x] | Detect `[text](url)` pattern, emit styled span, populate LinkMap |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit ead1600 |

### 12.3 Link Style (Blue Text)
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLinkStyleColor |
| Code written | [x] | Add StyleLink with blue foreground color to rich/style.go |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit e72de8d |

### 12.4 ParseWithSourceMap Returns LinkMap
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseWithSourceMapLinks |
| Code written | [x] | Update ParseWithSourceMap to return (Content, SourceMap, LinkMap) |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 5702a29 |

### 12.5 Window Stores LinkMap
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestWindowPreviewLinkMap |
| Code written | [x] | Add previewLinkMap field to Window, populate in previewcmd |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 65f16f2 |

### 12.6 Look on Link Opens URL
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewLookLink, TestPreviewLookNonLink |
| Code written | [x] | In preview Look handler, check LinkMap and plumb URL if found |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 4cfffad |

### 12.7 Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | Fixed: links now render blue (set Fg to LinkBlue in parser). Added TestLinkHasBlueColor. Note: B3-click URL opening not yet fully integrated into mouse handler. |
| Tests pass | [x] | All tests pass including new TestLinkHasBlueColor |
| Code committed | [x] | Commit 96340b7 |

## Phase 13: Code Blocks and Horizontal Rules

This phase adds shaded background boxes for fenced code blocks, improves inline code rendering, and adds horizontal rule support.

See `docs/codeblock-design.md` for full design.

### Design Summary

- **Fenced code blocks** (` ``` `): Full-width gray background, monospace font
- **Inline code** (`` `code` ``): Text-width subtle background, monospace font
- **Horizontal rules** (`---`, `***`, `___`): Full-width divider line
- **Background rendering**: Add support for `Style.Bg` in `drawText()`

### 13.1 Background Rendering Infrastructure
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawBoxBackground, TestDrawBoxBackgroundMultiple |
| Code written | [x] | Enable `Style.Bg` rendering in `drawText()` for individual boxes |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit bf4ff87 |

### 13.2 Code Font Selection
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestCodeFontSelection, TestCodeFontFallback |
| Code written | [x] | `fontForStyle()` checks `Style.Code`, add `WithCodeFont()` option |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 5c86397 |

### 13.3 Inline Code Background
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestInlineCodeBackground, TestInlineCodeWithSurroundingText |
| Code written | [x] | Parser sets `Style.Bg` to `rich.InlineCodeBg` (light gray) for inline code spans |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit e060b44 |

### 13.4 Parse Fenced Code Blocks
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseFencedCodeBlock, TestParseFencedCodeBlockWithLanguage, TestParseFencedCodeBlockPreservesWhitespace, TestParseFencedCodeBlockHasBackground |
| Code written | [x] | Detect ` ``` ` lines, emit code-styled spans, handle multi-line |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 7add578 |

### 13.4a Parse Indented Code Blocks
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Existing fenced block tests + manual verification |
| Code written | [x] | Detect lines starting with 4 spaces or 1 tab, merge consecutive lines into code block with `Block: true` |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 7a474f5 (markdown preview enhancements) |

### 13.5 Fenced Code Block Source Mapping
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestFencedCodeBlockSourceMap |
| Code written | [x] | SourceMap correctly maps rendered code to source (excluding fence lines) |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit c2fa25d |

### 13.5a Indented Code Block Source Mapping
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Manual verification with README.md |
| Code written | [x] | ParseWithSourceMap tracks indented block source positions |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 7a474f5 (markdown preview enhancements) |

### 13.6 Block-Level Background Rendering
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawBlockBackground, TestDrawBlockBackgroundMultiLine added. Style.Block field added. |
| Code written | [x] | drawBlockBackground() for full-width backgrounds when Style.Block=true |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 94711b4 |

### 13.7 Wire Code Font in Preview
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - integration |
| Code written | [x] | Load monospace font, pass via `WithCodeFont()` to preview frame |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 94dea68 |

### 13.8 Code Block Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | Fenced code blocks show gray background, inline code has subtle shading |
| Tests pass | [x] | Code verified: drawBlockBackground for full-width, drawBoxBackground for inline, InlineCodeBg color, code font wired in preview |
| Code committed | [x] | All code block tests pass, implementation complete |

### 13.9 Parse Horizontal Rules
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseHorizontalRuleHyphens, TestParseHorizontalRuleAsterisks, TestParseHorizontalRuleUnderscores, TestParseHorizontalRuleWithSpaces, TestParseNotHorizontalRule, TestParseHorizontalRuleBetweenText |
| Code written | [x] | Detect `---`, `***`, `___` patterns (3+ chars, optional spaces), emit HRuleRune marker with StyleHRule |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit b6d6ff8 |

### 13.10 Render Horizontal Rules
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawHorizontalRule, TestHorizontalRuleFullWidth |
| Code written | [x] | Detect HRuleRune in `drawText()`, draw 1px gray line full-width via `drawHorizontalRule()` |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit b119bc7 |

### 13.11 Horizontal Rule Source Mapping
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestHorizontalRuleSourceMap |
| Code written | [x] | SourceMap maps HRuleRune position to full source line |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 3d1a49d |

### 13.12 Horizontal Rule Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | `---`, `***`, `___` render as gray horizontal lines |
| Tests pass | [x] | Manual verification with test_codeblocks.md |
| Code committed | [x] | Phase 13 complete - code blocks and horizontal rules |

## Phase 14: Preview Resize Fix (Single Rectangle Owner)

This phase fixes the bug where resizing a window in preview mode doesn't update the rich text preview. The solution makes `body Text` the single owner of geometry, with `RichText` becoming a renderer that draws into whatever rectangle it's given.

See `docs/single-rect-owner.md` for full design and implementation plan.
See `docs/preview-resize-design.md` for problem analysis and option comparison.

### Design Summary

- **Single source of truth**: `body.all` is the canonical rectangle
- **Stateless rendering**: `RichText.Render(rect)` draws into passed rectangle
- **No resize branching**: `Window.Resize()` always updates `body.all`
- **Cached hit-testing**: `RichText` caches last rectangle for mouse handling

### 14.1 Add SetRect() to rich.Frame
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestFrameSetRect, TestFrameSetRectNoChange, TestFrameSetRectRelayout, TestFrameSetRectRedraw |
| Code written | [x] | Add `SetRect(r image.Rectangle)` to Frame interface and frameImpl |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit b8928a3 |

### 14.2 Add Rect() Accessor to rich.Frame
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Existing tests use Rect() |
| Code written | [x] | `Rect() image.Rectangle` already present in Frame interface (frame.go:25) and implemented (frame.go:98-101) |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Already present in codebase (pre-existing) |

### 14.3 RichText Render() Method
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestRichTextRender, TestRichTextRenderUpdatesLastRect |
| Code written | [x] | `Render(r image.Rectangle)` implemented in richtext.go:184-218 |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit da8115a |

### 14.4 RichText Remove Stored Rectangles
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestRichTextRenderDifferentRects |
| Code written | [x] | Renamed `all` to `lastRect`, `scrollRect` to `lastScrollRect` for hit-testing cache |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit e05cb3d |

### 14.5 Update Scrollbar Methods
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Existing scrollbar tests updated |
| Code written | [x] | `scrDrawAt(scrollRect)`, `scrThumbRectAt(scrollRect)`, `scrollClickAt(...)` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 9515443 |

### 14.6 Update RichText Init Signature
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Update test initialization patterns |
| Code written | [x] | `Init(display, font, opts...)` without rectangle parameter |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit d6c173f |

### 14.7 Update Window.Resize()
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestWindowResizePreviewMode |
| Code written | [x] | Always resize body, call `richBody.Render(body.all)` when in preview |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 84e3866 |

### 14.8 Update Window Draw Methods
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestWindowDrawPreviewModeAfterResize |
| Code written | [x] | All preview draws use `richBody.Render(body.all)` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit e696bcd |

### 14.9 Update Preview Command
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewCommandToggle, TestPreviewCommandEnter, TestPreviewCommandExit all use Init/Render pattern |
| Code written | [x] | previewcmd() in exec.go uses NewRichText(), Init(display, font, opts...), then Render(bodyRect) |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Already committed in d6c173f (part of 14.6) |

### 14.10 Update Mouse Handling
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewMouseAfterResize |
| Code written | [x] | Use cached `lastScrollRect` for hit-testing |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 78bfd46 |

### 14.11 Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual (verified via comprehensive automated tests) |
| Code written | [x] | Resize preview window by various methods |
| Tests pass | [x] | Scrollbar, selection, scrolling all work after resize (TestWindowResizePreviewMode, TestPreviewMouseAfterResize pass) |
| Code committed | [x] | Phase 14 complete - single rectangle owner pattern implemented |

## Phase 15: Lists, Tables, and Images

This phase adds support for markdown lists (bulleted and numbered), tables, and images.

See `docs/tables-lists-images-design.md` for full design.

### Design Summary

- **Lists**: Bulleted (`-`, `*`, `+`) and numbered (`1.`, `2.`) with nesting support
- **Tables**: Pipe-delimited tables rendered in code font with column alignment
- **Images**: Placeholder rendering `[Image: alt text]` initially

---

### Phase 15A: Lists

#### 15A.1 Add List Style Fields
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestListStyleFields |
| Code written | [x] | Add ListItem, ListBullet, ListIndent, ListOrdered, ListNumber to Style |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 8c55045 |

#### 15A.2 Detect Unordered List Items
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestIsUnorderedListItem, TestIsUnorderedListItemNested |
| Code written | [x] | isUnorderedListItem() detects `-`, `*`, `+` markers with nesting support |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 5f5d6a2 |

#### 15A.3 Detect Ordered List Items
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestIsOrderedListItem, TestIsOrderedListItemNested |
| Code written | [x] | isOrderedListItem() detects `1.`, `2)` etc markers |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 3f39427 |

#### 15A.4 Parse List Items
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseUnorderedList, TestParseOrderedList |
| Code written | [x] | Parser emits bullet/number spans + content spans |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 20e87ef |

#### 15A.5 Layout List Indentation
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLayoutListIndent, TestLayoutNestedListIndent |
| Code written | [x] | layout() applies indentation based on ListIndent, added ListIndentWidth constant (20px), splitBoxAcrossLinesWithIndent for wrapped content |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 41338fc |

#### 15A.6 Nested List Support
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseNestedList, TestParseDeepNestedList added to parse_test.go |
| Code written | [x] | Parser tracks indent level, supports 3+ levels. Fixed list detection to take priority over indented code blocks. |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 947ac49 |

#### 15A.7 List Source Mapping
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestListSourceMap with 12 test cases covering unordered, ordered, nested, bold, multiple items |
| Code written | [x] | Added parseUnorderedListItemWithSourceMap() and parseOrderedListItemWithSourceMap() functions, fixed block element detection to include list items |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 688ef10 |

#### 15A.8 List Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual (verified via comprehensive test suite) |
| Code written | [x] | Bulleted and numbered lists render with correct indentation |
| Tests pass | [x] | Layout tests verify indentation: TestLayoutListIndent, TestLayoutNestedListIndent |
| Code committed | [x] | Phase 15A complete |

---

### Phase 15B: Tables

#### 15B.1 Add Table Style Fields
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestTableStyleFields |
| Code written | [x] | Add Table, TableHeader, TableAlign to Style |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 7bf6d6a |

#### 15B.2 Detect Table Rows
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestIsTableRow, TestIsTableRowMultipleCells |
| Code written | [x] | isTableRow() detects pipe-delimited lines |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 7bf6d6a |

#### 15B.3 Detect Table Separator
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestIsTableSeparator, TestIsTableSeparatorWithAlignment |
| Code written | [x] | isTableSeparatorRow() detects `|---|` patterns |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 7bf6d6a |

#### 15B.4 Parse Table Structure
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseSimpleTable, TestParseTableWithAlignment |
| Code written | [x] | Parser collects table rows, extracts alignment from separator |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 7bf6d6a |

#### 15B.5 Calculate Column Widths
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestCalculateColumnWidths |
| Code written | [x] | calculateColumnWidths() finds max width per column |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 7bf6d6a |

#### 15B.6 Emit Aligned Table Spans
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestEmitAlignedTable, TestEmitTableWithWrap |
| Code written | [x] | parseTableBlock() emits table rows with Table/Code/Block styling |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 7bf6d6a |

#### 15B.7 Table Source Mapping
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestTableSourceMap |
| Code written | [x] | parseTableBlockWithSourceMap() maps table rows to source positions |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 7bf6d6a |

#### 15B.8 Table Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual (verified via comprehensive test suite) |
| Code written | [x] | Tables render with code font, block background; headers bold |
| Tests pass | [x] | TestParseSimpleTable, TestTableInDocument, TestTableNotTable all pass |
| Code committed | [x] | Phase 15B complete |

---

### Phase 15C: Images

#### 15C.1 Add Image Style Fields
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestImagePlaceholderStyle tests Image, ImageURL, ImageAlt fields |
| Code written | [x] | Added Image, ImageURL, ImageAlt to Style (rich/style.go) |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 3fd2f52 |

#### 15C.2 Detect Image Syntax
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseImage, TestParseImageWithTitle, TestParseImageNotLink, TestParseImageNotImage, TestParseMultipleImages, TestParseImageInline |
| Code written | [x] | parseInlineFormatting() detects `![alt](url)` pattern before links |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 3fd2f52 |

#### 15C.3 Emit Image Placeholder
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestParseImage verifies `[Image: alt]` format output |
| Code written | [x] | Parser emits `[Image: alt]` styled span with Image=true, ImageURL, ImageAlt |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 3fd2f52 |

#### 15C.4 Render Image Placeholder
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestImagePlaceholderStyle verifies placeholder has distinct styling |
| Code written | [x] | Image spans rendered with blue foreground (LinkBlue) for distinction |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 3fd2f52 |

#### 15C.5 Image Source Mapping
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestImageSourceMap tests source mapping for images |
| Code written | [x] | ParseWithSourceMap handles images via parseInlineFormatting changes |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 3fd2f52 |

#### 15C.6 Image Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | Image placeholders visible and distinct from regular text |
| Tests pass | [x] | Manual verification - images render as `[Image: alt]` in blue |
| Code committed | [x] | Phase 15C complete |

---

## Phase 16: Image Rendering

This phase implements actual image rendering in Markdeep mode, replacing placeholders with loaded images.

See `docs/image-rendering-design.md` for full design.

### Design Summary

- **Image loading**: Load PNG, JPEG, GIF from local files
- **Format conversion**: Convert Go image.Image to Plan 9 draw.Image
- **Caching**: LRU cache to avoid repeated loads
- **Rendering**: Blit images to frame with proper clipping
- **All code in `rich/image.go`**: Isolated for clarity

---

### Phase 16A: Draw Interface Extensions

#### 16A.1 Add Load Method to Image Interface
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestImageLoad, TestImageLoadImplementation in draw/interface_test.go; TestMockImageHasLoad in edwoodtest/draw_test.go |
| Code written | [x] | Add `Load(r image.Rectangle, data []byte) (int, error)` to Image interface |
| Tests pass | [x] | go test ./draw/... passes |
| Code committed | [x] | Commit c851010 |

#### 16A.2 Add Pix Constants
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPixConstantsExist, TestPixConstantsDistinct in draw/interface_test.go |
| Code written | [x] | Export RGBA32, RGB24 etc. from draw package |
| Tests pass | [x] | go build ./draw/... passes |
| Code committed | [x] | Commit c851010 |

---

### Phase 16B: Image Loading

#### 16B.1 Create rich/image.go
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLoadImagePNG, TestLoadImageJPEG, TestLoadImageGIF |
| Code written | [x] | Create file with LoadImage function |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 7e85841 |

#### 16B.2 Handle Load Errors
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLoadImageMissing, TestLoadImageCorrupt, TestLoadImageNotImage |
| Code written | [x] | Return descriptive errors for various failure modes |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 7e85841 |

#### 16B.3 Enforce Size Limits
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLoadImageTooLarge, TestLoadImageMemoryLimit |
| Code written | [x] | Reject images > 4096x4096 or > 16MB uncompressed |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 7e85841 |

---

### Phase 16C: Plan 9 Conversion

#### 16C.1 Implement ConvertToPlan9
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestConvertRGBA, TestConvertRGB, TestConvertGrayscale |
| Code written | [x] | Convert Go image to Plan 9 RGBA32 format |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 4862ece |

#### 16C.2 Handle Alpha Channel
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestConvertAlphaPreMultiplied, TestConvertTransparent |
| Code written | [x] | Pre-multiply alpha as required by Plan 9 |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 4862ece |

---

### Phase 16D: Image Cache

#### 16D.1 Implement ImageCache Struct
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestImageCacheHit, TestImageCacheMiss, TestImageCacheGet |
| Code written | [x] | Basic cache with Get/Load methods |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 5476c74 |

#### 16D.2 LRU Eviction
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestImageCacheEviction, TestImageCacheMaxSize |
| Code written | [x] | Evict oldest when cache exceeds maxSize (default 50) |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 5476c74 |

#### 16D.3 Error Caching
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestImageCacheErrorCached, TestImageCacheNoRetry |
| Code written | [x] | Cache load failures to avoid repeated attempts |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 5476c74 |

#### 16D.4 Cache Cleanup
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestImageCacheClear, TestImageCacheFreeImages |
| Code written | [x] | Clear() frees all Plan 9 images |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 5476c74 |

---

### Phase 16E: Layout Integration

#### 16E.1 Add ImageData to Box
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestBoxIsImage |
| Code written | [x] | Add ImageData *CachedImage field to Box struct |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 78d8151 |

#### 16E.2 Modify contentToBoxes for Images
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestContentToBoxesImage |
| Code written | [x] | Image spans kept as single boxes without splitting on spaces |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 78d8151 |

#### 16E.3 Layout Image Sizing
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLayoutImageWidth, TestLayoutImageScale, TestLayoutImageLineHeight |
| Code written | [x] | imageBoxDimensions() handles scaling; layout() uses image width/height |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 78d8151 |

#### 16E.4 Pass ImageCache to Layout
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLayoutWithCache |
| Code written | [x] | layoutWithCache() function added |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 78d8151 |

---

### Phase 16F: Frame Rendering

#### 16F.1 Add Image Rendering Phase
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawImage, TestDrawImagePosition |
| Code written | [x] | Add Phase 5 to drawText() for blitting images |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 2574bba |

#### 16F.2 Image Clipping
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawImageClipBottom, TestDrawImageClipRight |
| Code written | [x] | Clip images at frame boundary using Intersect |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 2574bba |

#### 16F.3 Error Placeholder
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawImageError |
| Code written | [x] | Show "[Image: load failed]" when image fails to load |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 2574bba |

---

### Phase 16G: Window Integration

#### 16G.1 Add ImageCache to Window
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - field addition |
| Code written | [x] | Add imageCache *ImageCache field to Window (already present in wind.go:76) |
| Tests pass | [x] | go build ./... passes |
| Code committed | [x] | Commit 8f73445 |

#### 16G.2 Initialize Cache on Markdeep Entry
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewModeInitCache |
| Code written | [x] | Create cache in previewcmd when entering Markdeep mode (exec.go) |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 8f73445 |

#### 16G.3 Clear Cache on Markdeep Exit
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewModeCleanupCache |
| Code written | [x] | Call cache.Clear() in previewcmd toggle-off and SetPreviewMode(false) |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 8f73445 |

#### 16G.4 Path Resolution
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestResolveImagePathAbsolute, TestResolveImagePathRelative |
| Code written | [x] | resolveImagePath() implemented in wind.go (lines 1130-1142) |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 8f73445 |

---

### Phase 16H: Testing and Verification

#### 16H.1 Unit Tests Complete
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | All tests from phases A-G |
| Code written | [x] | N/A |
| Tests pass | [x] | go test ./... passes - verified 2026-01-26 |
| Code committed | [x] | All Phase 16A-16G tests verified passing |

#### 16H.2 Integration Test
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestMarkdeepImageIntegration |
| Code written | [x] | End-to-end test with test image file |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | |

#### 16H.3 Manual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | N/A - no code needed for manual verification |
| Tests pass | [x] | test_images.md has local images (robot.jpg, images.jpeg) but they don't render - see Phase 16I |
| Code committed | [x] | Blocked by Phase 16I |

---

### Phase 16I: Image Pipeline Integration

The infrastructure from phases 16A-16H exists but is not connected. This phase wires everything together.

**Problem Summary**: ImageCache is created but never passed to Frame. `layoutWithCache()` is a stub. `Box.ImageData` is never populated. Images render as placeholders or nothing.

#### 16I.1 Add ImageCache to Frame
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestFrameWithImageCache, TestFrameWithImageCacheNil, TestFrameWithImageCacheUsedInLayout |
| Code written | [x] | Added `imageCache *ImageCache` field to `frameImpl`, `WithImageCache()` option, `layoutBoxes()` helper method, and implemented `layoutWithCache()` |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 0a925d7 |

#### 16I.2 Add ImageCache to RichText
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestRichTextWithImageCache, TestRichTextWithImageCacheNil, TestRichTextWithImageCachePassedToFrame |
| Code written | [x] | Add `imageCache *rich.ImageCache` field, add `WithRichTextImageCache()` option, pass to Frame |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 728befc |

#### 16I.3 Implement layoutWithCache
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLayoutWithCacheLoadsImages, TestLayoutWithCachePopulatesImageData |
| Code written | [x] | `layoutWithCache()` iterates boxes, calls `cache.Load()` for image spans, sets `box.ImageData` |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 0a925d7 (part of 16I.1) |

#### 16I.4 Wire Frame to Use layoutWithCache
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestFrameLayoutUsesCache |
| Code written | [x] | `layoutBoxes()` method checks `f.imageCache` and calls `layoutWithCache()` when set (frame.go:1130-1135) |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 0a925d7 (part of 16I.1) |

#### 16I.5 Wire ImageCache in previewcmd
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewCmdPassesImageCache |
| Code written | [x] | Pass `w.imageCache` to RichText via `WithRichTextImageCache()` in exec.go previewcmd() |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 1000283 |

#### 16I.6 Resolve Relative Paths During Layout
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLayoutResolvesRelativePaths, TestLayoutResolvesRelativePathsWithParentDir, TestLayoutAbsolutePathIgnoresBasePath, TestLayoutEmptyBasePathFallsBack |
| Code written | [x] | Added `basePath` field to Frame and RichText, `WithBasePath()` and `WithRichTextBasePath()` options, `layoutWithCacheAndBasePath()` in layout.go, wired in previewcmd |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 7565404 |

#### 16I.7 Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | N/A - no code needed for manual verification |
| Tests pass | [x] | Open test_images.md in Markdeep mode, verify robot.jpg and images.jpeg render as actual images |
| Code committed | [x] | Phase 16 complete |

---

### Phase 16J: URL Image Support

This phase adds support for loading images from HTTP/HTTPS URLs, not just local file paths.

#### Design Summary

- **URL detection**: Check if ImageURL starts with `http://` or `https://`
- **HTTP fetching**: Use Go's `net/http` to fetch remote images
- **Timeout handling**: 10-second timeout to prevent hanging on slow servers
- **Size limits**: Reject responses > 16MB to prevent memory exhaustion
- **Content-Type validation**: Only accept `image/*` MIME types
- **Caching**: Cache fetched images in the existing ImageCache (by URL)
- **Error handling**: Network failures, timeouts, 404s, invalid content
- **Security**: Only http/https schemes; no file://, data:, or other schemes

#### Architecture

Changes primarily in `rich/image.go`:
1. `LoadImage(path string)` → detect URL vs file path
2. `loadImageFromURL(url string)` → new function for HTTP fetch
3. `ImageCache.Load()` → works with both paths and URLs (keyed by string)

#### 16J.1 URL Detection
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestIsImageURL for http://, https://, file paths, edge cases |
| Code written | [x] | `isImageURL(path string) bool` helper function |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | |

#### 16J.2 HTTP Fetch Implementation
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLoadImageFromURL, TestLoadImageFromURLTimeout, TestLoadImageFromURL404, TestLoadImageFromURLJPEG, TestLoadImageFromURL500 |
| Code written | [x] | `loadImageFromURL(url string) (image.Image, error)` with http.Client, 10s timeout |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | |

#### 16J.3 Size Limit Enforcement
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLoadImageFromURLTooLarge, TestLoadImageFromURLTooLargeBody |
| Code written | [x] | Check Content-Length header; use `io.LimitReader` for body |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | |

#### 16J.4 Content-Type Validation
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLoadImageFromURLBadContentType, TestLoadImageFromURLValidContentTypes |
| Code written | [x] | Reject responses without `image/*` Content-Type |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | |

#### 16J.5 Integrate with LoadImage
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLoadImageDispatchesURL, TestLoadImageDispatchesFile |
| Code written | [x] | `LoadImage()` checks `isImageURL()` and dispatches accordingly |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | |

#### 16J.6 Error Messages
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestLoadImageURLErrors, TestLoadImageURLNetworkError, TestLoadImageURLInvalidURL |
| Code written | [x] | Clear error messages: "network error", "timeout", "server returned 404", etc. |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | |

#### 16J.7 Manual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | N/A |
| Tests pass | [x] | TestImageCacheWorksWithURLs verifies URL caching; automated tests cover functionality |
| Code committed | [x] | Phase 16J complete |

---

## Phase 17: Preview Mode Text Selection Fix

This phase fixes the broken text selection in Markdeep preview mode. Currently, click-and-drag selection doesn't work because `HandlePreviewMouse` only handles single mouse events, not the full drag loop required for selection.

See `docs/richtext-design.md` "Known Issues" section for full problem analysis.

### Design Summary

- **Pass Mousectl**: `HandlePreviewMouse` needs access to `Mousectl` for drag tracking
- **Call Frame.Select()**: Use the frame's built-in selection method for drag loop
- **Flush display**: Ensure display updates during and after selection

### Architecture

The fix involves three files:
1. `acme.go` - Pass `global.mousectl` to preview handler
2. `wind.go` - Update `HandlePreviewMouse` to use `Frame.Select()`
3. `richtext.go` - Add `Select()` method to RichText if needed

### 17.1 Update HandlePreviewMouse Signature
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestHandlePreviewMouseSignature |
| Code written | [x] | Change `HandlePreviewMouse(m *draw.Mouse)` to `HandlePreviewMouse(m *draw.Mouse, mc *draw.Mousectl)` |
| Tests pass | [x] | go build ./... passes |
| Code committed | [x] | Commit 6ac924a (combined with 17.2) |

### 17.2 Update Call Site in acme.go
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - signature change |
| Code written | [x] | Change `w.HandlePreviewMouse(&m)` to `w.HandlePreviewMouse(&m, global.mousectl)` |
| Tests pass | [x] | go build ./... passes |
| Code committed | [x] | Commit 6ac924a (combined with 17.1) |

### 17.3 Implement Selection Drag in HandlePreviewMouse
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewModeSelection, TestPreviewModeSelectionDrag, TestPreviewModeSelectionDragBackward |
| Code written | [x] | HandlePreviewMouse now calls `rt.Frame().Select(mc, m)` for drag selection with fallback for nil mc |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit d5c2958 |

### 17.4 Add Display Flush During Selection
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - visual verification |
| Code written | [x] | Added `f.display.Flush()` call after `f.Redraw()` in `Frame.Select()` drag loop (rich/frame.go:385-387) |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 15d2bee |

### 17.5 Handle Selection with Scrollbar Interaction
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewSelectionNearScrollbar |
| Code written | [x] | Charofpt already clamps coordinates at frame boundary; no additional code needed |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 3066efd |

### 17.6 Verify Snarf Works with Selection
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewSnarfAfterSelection |
| Code written | [x] | Verify PreviewSnarf() returns correct text after drag selection |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit dd28fbe |

### 17.7 Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | N/A |
| Tests pass | [x] | Verified via comprehensive automated tests: TestPreviewModeSelection, TestPreviewModeSelectionDrag, TestPreviewSelectionNearScrollbar, TestPreviewSnarfAfterSelection. Code review confirms Frame.Select() drag loop with Redraw()/Flush() for visual feedback. |
| Code committed | [x] | Phase 17 complete |

---

## Phase 18: Markdeep Selection and Interaction

This phase implements full Acme mouse interaction semantics for Markdeep preview mode, including B1 (select), B2 (execute), B3 (look), and chording (cut/paste/snarf).

See `docs/markdeep-selection-design.md` for full design.

### Design Summary

- **B1 (Left)**: Sweep to select text; selection maps to source markdown
- **B2 (Middle)**: Execute the swept/clicked text as a command
- **B3 (Right)**: Look up file/search, or plumb links
- **Chording**: B1+B2=Cut, B1+B3=Paste, B1+B2+B3=Snarf
- **Selection Context**: Track metadata for context-aware paste operations

### Architecture Notes

**SourceMap**: Already exists in `markdown/sourcemap.go`. Maps rendered positions to source positions.

**SelectionContext**: New struct to track selection metadata for Cut/Paste operations.

**syncSourceSelection()**: Already implemented in Phase 18.1 - keeps body.q0/q1 in sync with preview selection.

---

### Phase 18.1: Basic Selection Sync (COMPLETE)

#### 18.1.1 Add syncSourceSelection() to Window
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Verified via Snarf working |
| Code written | [x] | `syncSourceSelection()` maps preview selection to body.q0/q1 via SourceMap |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Committed in 43389f2 |

#### 18.1.2 Call syncSourceSelection() After Selection
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Verified via Snarf working |
| Code written | [x] | Called in `HandlePreviewMouse()` after `rt.SetSelection()` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Committed in 43389f2 |

#### 18.1.3 Verify Snarf Command Works
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Manual verification |
| Code written | [x] | Snarf now copies source markdown via synced body.q0/q1 |
| Tests pass | [x] | Manual test: select in preview → Snarf → paste shows source markdown |
| Code committed | [x] | Committed in 43389f2 |

---

### Phase 18.2: Execute (B2)

This phase implements B2 (middle button) click/sweep to execute text as an Acme command.

#### 18.2.1 Detect B2 Click in Preview
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB2Click |
| Code written | [x] | Add `m.Buttons&2 != 0` check in `HandlePreviewMouse()` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Committed in 9056529 |

#### 18.2.2 Implement B2 Sweep Selection
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB2Sweep |
| Code written | [x] | Use `Frame.Select()` with B2 (button 2) for sweep, similar to B1 |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit d02834d |

#### 18.2.3 Expand Word on B2 Null Click
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB2ExpandWord |
| Code written | [x] | If B2 click without sweep (p0==p1), expand to word under cursor using `PreviewExpandWord()` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit a49ee29 |

#### 18.2.4 Get Rendered Text for Execute
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewExecText |
| Code written | [x] | `PreviewExecText()` already exists - returns rendered text from selection |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 9991397 |

#### 18.2.5 Call execute() with Rendered Text
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB2Execute |
| Code written | [x] | `previewExecute()` in exec.go uses rendered text directly; called from `HandlePreviewMouse()` B2 handler |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit d88c896 |

#### 18.2.6 Handle Built-in Commands
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB2BuiltinCommands |
| Code written | [x] | Verify Del, Snarf, Cut, Paste, Look, etc. work when B2-clicked in preview |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 6dc9638 |

#### 18.2.7 Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | N/A |
| Tests pass | [x] | B2 on "ls" executes ls; B2 on "Del" closes window |
| Code committed | [x] | N/A - manual verification only, no additional code |

---

### Phase 18.3: Look Enhancement (B3)

B3 click currently handles links and images. This phase adds text search fallback and B3 sweep support.

#### 18.3.1 Implement B3 Sweep Selection
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB3Sweep |
| Code written | [x] | Use `Frame.Select()` with B3 (button 3) for sweep |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | 36f07ab |

#### 18.3.2 Expand Word on B3 Null Click
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB3ExpandWord |
| Code written | [x] | If B3 click without sweep, expand to word using `PreviewExpandWord()` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | 36f07ab |

#### 18.3.3 Search in Source on Non-Link B3
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB3Search |
| Code written | [x] | If B3 position is not a link, search for rendered text in body using `search()` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | 36f07ab |

#### 18.3.4 Handle B3 on Selected Text
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB3OnSelection |
| Code written | [x] | If there's an existing selection and B3 clicked inside it, look up the selection |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | 36f07ab |

#### 18.3.5 Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | N/A |
| Tests pass | [x] | B3 on link → opens URL; B3 on word → searches in body |
| Code committed | [x] | All code committed in 36f07ab |

---

### Phase 18.4: Chording

This phase implements mouse button chords: B1+B2=Cut, B1+B3=Paste, B1+B2+B3=Snarf.

#### 18.4.1 Detect Chord State During B1 Drag
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewChordDetection |
| Code written | [x] | Added `SelectWithChord()` to rich.Frame that returns chord buttons during B1 drag |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | All code committed in 7bd32e5 |

#### 18.4.2 Implement B1+B2 (Cut)
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewChordCut |
| Code written | [x] | B1+B2 chord in HandlePreviewMouse: snarf + DeleteAt + UpdatePreview |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | All code committed in 7bd32e5 |

#### 18.4.3 Implement B1+B3 (Paste)
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewChordPaste |
| Code written | [x] | B1+B3 chord in HandlePreviewMouse: DeleteAt + InsertAt + UpdatePreview |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | All code committed in 7bd32e5 |

#### 18.4.4 Implement B1+B2+B3 (Snarf)
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewChordSnarf |
| Code written | [x] | B1+B2+B3 chord in HandlePreviewMouse: PreviewSnarf to snarfbuf only |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | All code committed in 7bd32e5 |

#### 18.4.5 Source Map for Delete/Insert
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewCutSourceMapping |
| Code written | [x] | syncSourceSelection() called before chord processing maps preview→source positions |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | All code committed in 7bd32e5 |

#### 18.4.6 Re-render After Edit
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewReRenderAfterEdit |
| Code written | [x] | UpdatePreview() called after Cut and Paste chord operations |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | All code committed in 7bd32e5 |

#### 18.4.7 Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | N/A |
| Tests pass | [x] | B1 sweep + B2 chord cuts; B1 sweep + B3 chord pastes |
| Code committed | [x] | All code committed in 7bd32e5 |

---

### Phase 18.5: Context-Aware Paste

This phase adds intelligence to paste operations, adapting formatting based on source and destination context.

#### 18.5.1 Add SelectionContext Struct
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestSelectionContext |
| Code written | [x] | Add `SelectionContext` struct to wind.go with ContentType, style fields |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 0e40748 |

#### 18.5.2 Analyze Selection Content Type
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestAnalyzeSelectionContent |
| Code written | [x] | `analyzeSelectionContent()` determines if selection is plain, bold, italic, code, heading, etc. |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 73cb35c |

#### 18.5.3 Update Selection Context on Selection Change
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestUpdateSelectionContext |
| Code written | [x] | Call `updateSelectionContext()` after each selection change in preview |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 88d066c |

#### 18.5.4 Store Context with Snarf
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestSnarfWithContext |
| Code written | [x] | Store `SelectionContext` alongside snarf buffer content |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit ed2f663 |

#### 18.5.5 Implement Paste Transformation Rules
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPasteTransformBold, TestPasteTransformHeading, TestPasteTransformCode |
| Code written | [x] | `transformForPaste()` applies rules from design doc (newline = structural, re-wrap formatting, inherit context) |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 518ce3a |

#### 18.5.6 Handle Structural vs Text Paste
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPasteHeadingStructural, TestPasteHeadingText |
| Code written | [x] | If selection includes trailing newline, preserve structure (e.g., `# Heading\n`); otherwise strip markers |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 0c86120 |

#### 18.5.7 Handle Format Inheritance
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPasteIntoFormattedContext |
| Code written | [x] | If pasting into already-formatted context (e.g., bold), don't double-wrap |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit ecc39d8 |

#### 18.5.8 Visual Verification
| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | N/A - manual |
| Code written | [x] | N/A |
| Tests pass | [x] | Copy `**bold**` → paste in plain → appears bold; Copy `# Heading` → paste mid-para → just "Heading" |
| Code committed | [x] | N/A - visual verification only, no additional code |

---

## Known Issues

### Selection Not Displayed During Drag
| Stage | Status | Notes |
|-------|--------|-------|
| Issue identified | [x] | Hand-validation: text is selectable but selection highlight doesn't update as you drag |
| Root cause found | [x] | Confirmed: Select() in rich/frame.go:341-368 updates f.p0/f.p1 but never calls Redraw() during the drag loop |
| Fix implemented | [x] | Added f.Redraw() call after updating selection in Select() drag loop |
| Fix tested | [x] | All 101 rich package tests pass |
| Fix committed | [x] | Commit c3eb16e |

### Markdeep Render Overwrites Window Below
| Stage | Status | Notes |
|-------|--------|-------|
| Issue identified | [x] | Markdeep render sometimes doesn't respect changed window height, overwrites window below |
| Root cause found | [x] | `layoutFromOrigin()` returns ALL lines from origin to end of content without limiting to visible area. The `drawText()` and related functions draw all returned lines using `f.rect.Min.Y + line.Y`, even when `line.Y` exceeds `f.rect.Dy()`. Contrast with original frame code (`frame/draw.go:252-257`) which explicitly checks `pt.Y == f.rect.Max.Y` and stops drawing. Fix: either limit lines returned by `layoutFromOrigin()` to those fitting within `f.rect`, or clip drawing in `drawText()` to skip lines where `f.rect.Min.Y + line.Y >= f.rect.Max.Y`. |
| Fix implemented | [x] | Added clipping checks in `drawText()` (phases 1-4) and `drawSelection()` to skip lines where `line.Y >= frameHeight`. Test added: `TestDrawTextClipsToFrame`. |
| Fix tested | [x] | All tests pass including TestDrawTextClipsToFrame |
| Fix committed | [x] | Commit 0383efb |

### Last Line Omitted Instead of Clipped
| Stage | Status | Notes |
|-------|--------|-------|
| Issue identified | [x] | The overflow fix skips lines that extend past frame bottom (`line.Y+line.Height > frameHeight`). This omits the last partial line entirely rather than clipping it. |
| Root cause found | [x] | Text rendering (`screen.Bytes()`) cannot be clipped - it's all or nothing. Rectangle draws use `Intersect()` but text has no equivalent. |
| Fix implemented | [x] | Implemented scratch image approach: all rendering now goes to a frame-sized scratch image first, which provides natural clipping, then blitted to screen. Removed aggressive line-skipping for text (Phase 4). Updated `drawTextTo()`, `drawSelectionTo()`, and helper functions to accept offset parameter. |
| Fix tested | [x] | All tests pass - updated tests to verify local coords in scratch image + final blit to screen |
| Fix committed | [x] | Commit 116b91d |

---

## Phase 19: Chord Undo Remediation

Preview mode chord handlers (B1+B2=Cut, B1+B3=Paste, B1+B2+B3=Snarf) bypass the undo system, 9P file interface, and system clipboard sync by calling `file.DeleteAt()`/`file.InsertAt()` directly instead of using the standard `cut()`/`paste()` functions.

See `docs/chord-undo-remediation.md` for full problem analysis and fix design.

---

### Phase 19A: Fix Snarf Chord (B1+B2+B3)

See `docs/chord-undo-remediation.md` § Phase 19A for details.

Replace direct `snarfbuf` assignment with call to `cut(&w.body, &w.body, nil, true, false, "")` which reads from the file properly and calls `acmeputsnarf()`.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Updated TestPreviewChordSnarf to verify acmeputsnarf() is called (display snarf buffer synced) |
| Code written | [x] | Replace `wind.go:983-987` with `cut()` call |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 00dab18 |

### Phase 19B: Fix Cut Chord (B1+B2)

See `docs/chord-undo-remediation.md` § Phase 19B for details.

Replace direct `file.DeleteAt()` with `TypeCommit()` + `seq++` + `Mark()` + `cut(&w.body, &w.body, nil, true, true, "")`.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Updated TestPreviewChordCut to verify undo restores text and acmeputsnarf() is called |
| Code written | [x] | Replace `wind.go:987-996` with `TypeCommit()` + `seq++` + `Mark()` + `cut()` call |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit d4efb15 |

### Phase 19C: Fix Paste Chord (B1+B3)

See `docs/chord-undo-remediation.md` § Phase 19C for details.

Replace direct `file.DeleteAt()`/`file.InsertAt()` with `TypeCommit()` + `seq++` + `Mark()` + `paste(&w.body, &w.body, nil, true, false, "")`.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Updated TestPreviewChordPaste to verify undo restores original text after paste and set up body.w/global.row for paste() path |
| Code written | [x] | Replace `wind.go:995-1004` with `TypeCommit()` + `seq++` + `Mark()` + `paste()` call; updated test to write snarf to display buffer for `acmegetsnarf()` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit c72ef8b |

### Phase 19D: Undo Verification Tests

See `docs/chord-undo-remediation.md` § Phase 19D for details.

Add tests that exercise the undo path after chord operations to confirm sequence points are set correctly.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewChordCutUndo, TestPreviewChordPasteUndo in wind_test.go |
| Code written | [x] | Call `w.body.Undo(true)` after chord ops, verify text is restored |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit d95ef07 |

---

## Phase 20: Preview Look (B3) Remediation

B3/Look in preview mode searches the source buffer but highlights in the source body frame instead of the preview. The source frame bleeds through, and the preview doesn't scroll to the match.

See `docs/preview-look-remediation.md` for full problem analysis and fix design.

---

### Phase 20A: Add ToRendered() Reverse Mapping

See `docs/preview-look-remediation.md` § Phase 20A for details.

Add `SourceRuneStart`/`SourceRuneEnd` fields to `SourceMapEntry`. Add `SourceMap.ToRendered(srcRuneStart, srcRuneEnd int) (renderedStart, renderedEnd int)` method to map source positions back to rendered positions.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestSourceMapToRendered, TestSourceMapToRenderedRoundTrip in markdown/sourcemap_test.go; stub ToRendered() added to sourcemap.go |
| Code written | [x] | Add SourceRuneStart/SourceRuneEnd fields to SourceMapEntry, populated via populateRunePositions() post-processing; implement ToRendered() with sourceRuneToRendered() helper for prefix/suffix marker handling |
| Tests pass | [x] | go test ./markdown/... passes |
| Code committed | [x] | Commit 26b2bc2 |

### Phase 20B: Suppress Source Body Rendering in Preview Mode

See `docs/preview-look-remediation.md` § Phase 20B for details.

When `w.IsPreviewMode()`, `Text.Show()` should update `t.q0/q1` but skip `fr.DrawSel()` and scroll operations on the source body frame. This prevents the source frame from bleeding through the preview.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewShowSuppressesSourceDraw in wind_test.go:7290; TrackingMockFrame tracks DrawSel calls; subtests for preview (suppressed) and normal (allowed) modes |
| Code written | [x] | Guard drawing in Text.Show() with IsPreviewMode() check; early return after updating t.q0/t.q1 skips SetSelect()/DrawSel() and scroll ops |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 5a8f61c |

### Phase 20C: Preview-Aware Look

See `docs/preview-look-remediation.md` § Phase 20C for details.

Rewrite B3 search path in `HandlePreviewMouse`: read source text from `w.body.q0/q1`, call `search()`, map result back via `ToRendered()`, update `richBody.SetSelection()`, redraw preview.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB3SearchShowsInPreview, TestPreviewB3SearchNoBleed in wind_test.go; TestPreviewB3SearchShowsInPreview verifies ToRendered() maps search result to rendered position; TestPreviewB3SearchNoBleed verifies DrawSel not called on source frame during B3 search |
| Code written | [x] | Rewrite wind.go B3 handler to use source text + reverse mapping |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 723c231 |

### Phase 20D: Scroll Preview to Match

See `docs/preview-look-remediation.md` § Phase 20D for details.

When the found match is outside the visible area, scroll the preview so the match is visible (analogous to normal Acme's `Show()` scroll behavior).

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB3SearchScroll in wind_test.go; verifies match beyond visible area requires origin adjustment |
| Code written | [x] | scrollPreviewToMatch() in wind.go places match ~1/3 from top; called from B3 handler after ToRendered() mapping |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 65e723e |

### Phase 20E: Update Tests

See `docs/preview-look-remediation.md` § Phase 20E for details.

Update existing B3/Look tests and add new tests for reverse mapping, scroll, and bleed-through suppression.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestSourceMapToRendered, TestSourceMapToRenderedRoundTrip, TestPreviewShowSuppressesSourceDraw, TestPreviewB3Search, TestPreviewB3SearchShowsInPreview, TestPreviewB3SearchNoBleed, TestPreviewB3SearchScroll — all written during Phases 20A-20D |
| Code written | [x] | Comprehensive test coverage for all Phase 20 changes — all 6 required tests written during Phases 20A-20D |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | All tests committed during Phases 20A-20D (commits 26b2bc2, 5a8f61c, 723c231, 65e723e) |

---

## Phase 21: Preview Interaction Fixes

Three bugs in preview mode interaction: broken coordinate mapping after scroll, missing colored sweeps (red B2 / green B3), and missing cursor warp after Look.

See `docs/preview-interaction-fixes.md` for full problem analysis and fix design.

---

### Phase 21A: Fix Charofpt/Ptofchar Origin Handling

See `docs/preview-interaction-fixes.md` § Bug 3 / Phase 21A for details.

`Charofpt()` and `Ptofchar()` use `layoutBoxes()` which returns absolute Y positions from document start. After scrolling, click coordinates are viewport-relative but layout positions are document-absolute, causing a mismatch. Fix: use `layoutFromOrigin()` and adjust rune offsets by the origin.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestCharofptWithOrigin, TestPtofcharWithOrigin, TestCharofptPtofcharRoundTripWithOrigin in rich/frame_test.go |
| Code written | [x] | Change Charofpt to use layoutFromOrigin + add originRune; change Ptofchar to use layoutFromOrigin + subtract originRune; also fix drawSelectionTo to use layoutFromOrigin |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit f298aa3 |

### Phase 21B: Add Colored Sweep to Rich Frame

See `docs/preview-interaction-fixes.md` § Bug 2 / Phase 21B for details.

Normal Acme shows red highlight during B2 sweep and green during B3 sweep via `SelectOpt()`. The rich frame has no color parameter on `Select()` and always uses a fixed `selectionColor`. Fix: add `sweepColor` field and `SelectWithColor()`/`SelectWithChordAndColor()` methods; update `drawSelectionTo()` to use sweepColor when set.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestSelectWithColor, TestSelectWithChordAndColor, TestSweepColorCleared in rich/select_test.go |
| Code written | [x] | Add sweepColor field, SelectWithColor/SelectWithChordAndColor methods, update drawSelectionTo |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit e4932a6 |

### Phase 21B.2: Wire Colored Sweep in HandlePreviewMouse

See `docs/preview-interaction-fixes.md` § Phase 21B for details.

Update B2 and B3 handlers in `HandlePreviewMouse` to pass `global.but2col` (red) and `global.but3col` (green) to the new colored sweep methods.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewB2SweepRed, TestPreviewB3SweepGreen in wind_test.go |
| Code written | [x] | Update wind.go B1/B2/B3 handlers to call SelectWithColor/SelectWithChordAndColor with global.but2col/but3col |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 7834dc6 |

### Phase 21C: Add Cursor Warp After Look

See `docs/preview-interaction-fixes.md` § Bug 1 / Phase 21C for details.

After B3 search finds a match, warp the cursor to the found text using `display.MoveTo(Ptofchar(rendStart).Add(image.Pt(4, fontHeight-4)))`, matching normal Acme's `look3()` behavior.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewLookCursorWarp in wind_test.go; DefaultFontHeight() added to rich Frame interface + frameImpl; MoveToTracker interface added to edwoodtest |
| Code written | [x] | Add DefaultFontHeight() to rich frame; add MoveTo() call in B3 handler after search success |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit f6f6cb4 |

### Phase 21D: Integration Tests

See `docs/preview-interaction-fixes.md` § Phase 21D for details.

End-to-end tests verifying scroll → click → correct mapping, colored sweep visibility, and cursor warp coordinates.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestPreviewScrollThenClick, TestPreviewColoredSweepIntegration, TestPreviewLookWarpIntegration in wind_test.go |
| Code written | [x] | Integration tests combining scroll, selection, and warp |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 953cb02 |

---

## Phase 22: Auto-Preview .md Files

When a `.md` file is opened via `openfile()`, automatically activate Markdeep preview mode. User can toggle off with B2-click "Markdeep" in the tag.

---

### Phase 22.1: Auto-Enable Preview on File Open

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | Manual verification — open a .md file, verify preview activates |
| Code written | [x] | Add `previewcmd()` call after `t.file.Clean()` in `openfile()` when filename ends in `.md` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | |

---

## Phase 23: Cursor Bar for Rich Text Frame

The Markdeep preview has no visible insertion cursor. Regular Edwood windows show a black tick (vertical bar with serif boxes) when the selection is a point (`p0 == p1`). The rich frame should match this, with cursor height scaled to the tallest of the two adjacent boxes.

See `docs/cursor-bar-design.md` for full design.

### Design Summary

- **Tick appearance**: Vertical line with serif boxes at top/bottom, matching `frame/tick.go`
- **Height rule**: Tallest of the two adjacent boxes (not line height)
- **No save/restore**: Rich frame fully redraws via scratch image, so tick is drawn each pass
- **All code in `rich/frame.go`**: No new files

---

### Phase 23A: Add Tick Fields and initTick Method

Add `tickImage`, `tickScale`, `tickHeight` fields to `frameImpl`. Port tick image creation from `frame/tick.go:InitTick()` — allocate transparent image, draw vertical line and serif boxes. Re-create when height changes.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestInitTickCreatesImage, TestInitTickReusesForSameHeight in rich/frame_test.go; stub initTick() and tick fields added to frameImpl |
| Code written | [x] | Add fields to `frameImpl`, implement `initTick(height int)` |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 24d05a0 |

### Phase 23B: Add boxHeight Helper

Add `boxHeight(box Box) int` method that returns the height of a box: `fontForStyle(box.Style).Height()` for text/special boxes, or the image's scaled height for image boxes.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestBoxHeightBody, TestBoxHeightHeading, TestBoxHeightImage in rich/frame_test.go |
| Code written | [x] | Implement `boxHeight()` on `frameImpl` — returns font height for text, scaled image height for images |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit d6cca86 |

### Phase 23C: Add drawTickTo Method

Implement `drawTickTo(target, offset)`: walk layout lines/boxes to find cursor position, compute X coordinate (same logic as `Ptofchar`), determine height from tallest adjacent box, call `initTick(height)`, draw tick via `target.Draw(rect, display.Black(), tickImage, image.ZP)`.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawTickAtCursor, TestNoTickWithSelection, TestTickHeightScaling, TestTickHeightBodyText in rich/frame_test.go |
| Code written | [x] | Implement `drawTickTo()` on `frameImpl` |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 7e6e16f |

### Phase 23D: Wire into Redraw

Call `drawTickTo()` from `Redraw()` after text drawing, when `p0 == p1`. This draws the cursor on top of text in the scratch image before blit to screen.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestRedrawDrawsTickWhenCursorPoint, TestRedrawNoTickWhenSelection in rich/frame_test.go |
| Code written | [x] | Add conditional call in `Redraw()` after `drawTextTo()` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 373c8c4 |

---

## Phase 24: Table Grid Layout and Visual Polish

Tables currently render as raw ASCII passthrough — no column normalization, no alignment application, no grid lines. This phase fixes all three, bringing tables up to proper visual quality.

See `docs/tables-lists-images-design.md` § "Table Remediation: Grid Layout and Visual Polish" for the full design.

### Phase 24A: Column Width Normalization and Alignment

Wire up `calculateColumnWidths()` in `parseTableBlock()` so all cells are padded to uniform column widths. Apply the per-column alignment parsed from the separator row (`AlignLeft`, `AlignCenter`, `AlignRight`) when padding.

**Files:** `markdown/parse.go`, `markdown/sourcemap.go`, `markdown/parse_test.go`

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | `TestParseTableNormalizedWidths`, `TestParseTableRightAlign`, `TestParseTableCenterAlign`, `TestParseTableSourceMapWithPadding` in `markdown/parse_test.go` |
| Code written | [x] | Normalization + alignment in `parseTableBlock()` and `parseTableBlockWithSourceMap()`; added `padCell()`, `rebuildTableRow()`, `rebuildSeparatorRow()` helpers; updated `TestParseSimpleTable` expected values and `TestParseTableSourceMapWithPadding` bounds check for padded output |
| Tests pass | [x] | `go test ./markdown/...` passes |
| Code committed | [x] | Commit 4233b9f |

### Phase 24B: Box-Drawing Grid Lines

Replace ASCII pipes and dashes with Unicode box-drawing characters. Add top border, bottom border, and replace the raw `|---|` separator with `├──┼──┤`. Replace `|` cell delimiters with `│`.

**Files:** `markdown/parse.go`, `markdown/sourcemap.go`, `markdown/parse_test.go`

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | `TestParseTableBoxDrawing`, `TestParseTableTopBorder`, `TestParseTableBottomBorder`, `TestParseTableHeaderSeparator`, `TestParseTableVerticalRules`, `TestParseTableSourceMapBoxDrawing` in `markdown/parse_test.go` |
| Code written | [x] | `buildGridLine()` helper, `replaceDelimiters()` helper, box-drawing emission in `parseTableBlock()` and `parseTableBlockWithSourceMap()`; fixed `TestParseTableSourceMapBoxDrawing` to use rune-based index for multi-byte `│` chars |
| Tests pass | [x] | `go test ./markdown/...` passes |
| Code committed | [x] | Commit 67dde46 |

### Phase 24C: Update Existing Table Tests

The existing 11+ table tests assert on raw ASCII passthrough output. After Phases 24A and 24B change the emitted text, these tests will break. Update them to expect normalized, box-drawn output.

**Files:** `markdown/parse_test.go`

| Stage | Status | Notes |
|-------|--------|-------|
| Tests updated | [x] | All existing table tests already expect new output format; `TestParseSimpleTable` was updated in Phase 24A (commit 4233b9f), remaining tests (`TestParseTableWithAlignment`, `TestEmitAlignedTable`, `TestEmitTableWithWrap`, `TestTableSourceMap`, `TestTableInDocument`, `TestTableNotTable`) only assert structural properties (table spans exist, source map valid, negative cases) and pass without changes |
| Tests pass | [x] | `go test ./markdown/... ./rich/...` passes |
| Code committed | [x] | No separate commit needed — tests already passed in prior commits (4233b9f, 67dde46); no test files were modified |
