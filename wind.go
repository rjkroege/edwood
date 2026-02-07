package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sync"
	"time"

	"9fans.net/go/plumb"
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/frame"
	"github.com/rjkroege/edwood/internal/ui"
	"github.com/rjkroege/edwood/markdown"
	"github.com/rjkroege/edwood/rich"
	"github.com/rjkroege/edwood/util"
)

type Window struct {
	display draw.Display
	lk      sync.Mutex
	ref     Ref
	tag     Text
	body    Text
	r       image.Rectangle

	//	isdir      bool // true if this Window is showing a directory in its body.
	filemenu   bool
	autoindent bool
	showdel    bool

	id    int
	addr  Range
	limit Range

	nopen      [QMAX]byte // number of open Fid for each file in the file server
	nomark     bool
	wrselrange Range
	rdselfd    *os.File // temporary file for rdsel read requests

	col    *Column
	eventx *Xfid
	events []byte

	owner       int // TODO(fhs): change type to rune
	maxlines    int
	dirnames    []string
	widths      []int
	incl        []string
	ctrllock    sync.Mutex // used for lock/unlock ctl mesage
	ctlfid      uint32     // ctl file Fid which has the ctrllock
	dumpstr     string
	dumpdir     string
	utflastqid  int    // Qid of last read request (QWbody or QWtag)
	utflastboff uint64 // Byte offset of last read of body or tag
	utflastq    int    // Rune offset of last read of body or tag

	tagfilenameend     int
	tagfilenamechanged bool
	tagsetting         bool
	tagsafe            bool // What is tagsafe for?
	tagexpand          bool
	taglines           int
	tagtop             image.Rectangle

	editoutlk chan bool

	// Preview mode fields for rich text rendering
	previewMode        bool                // true when showing rendered markdown preview
	richBody           *RichText           // rich text renderer for preview mode
	previewSourceMap   *markdown.SourceMap // maps rendered positions to source positions
	previewLinkMap     *markdown.LinkMap   // maps rendered positions to link URLs
	previewUpdateTimer *time.Timer         // debounce timer for preview updates
	imageCache         *rich.ImageCache    // cache for loaded images in preview mode
	selectionContext   *SelectionContext   // context metadata for the current preview selection

	// Preview double-click state (mirrors clicktext/clickmsec in text.go)
	previewClickPos  int        // rune position of last B1 null-click
	previewClickMsec uint32     // timestamp of last B1 null-click
	previewClickRT   *RichText  // which richtext received the last click

	// Incremental preview update state
	prevBlockIndex *markdown.BlockIndex  // block boundaries from last parse
	pendingEdits   []markdown.EditRecord // edits since last UpdatePreview
}

var (
	_ file.TagStatusObserver = (*Window)(nil) // Enforce at compile time that Window implements BufferObserver
	_ file.BufferObserver    = (*Window)(nil) // Enforce at compile time that TagIndex implements BufferObserver
)

func NewWindow() *Window {
	return &Window{}
}

// Initialize the headless parts of the window.
func (w *Window) initHeadless(clone *Window) *Window {
	w.tag.w = w
	w.taglines = 1
	w.tagsafe = false
	w.tagexpand = true
	w.body.w = w
	w.incl = []string{}
	global.WinID++
	w.id = global.WinID
	w.ref.Inc()
	if global.globalincref {
		w.ref.Inc()
	}

	w.ctlfid = MaxFid
	w.utflastqid = -1

	// Tag setup.
	f := file.MakeObservableEditableBuffer("", nil)

	if clone != nil {
		// TODO(rjk): Support something nicer like initializing from a Reader.
		// (Can refactor ObservableEditableBuffer.Load perhaps.
		clonebuff := make([]rune, clone.tag.Nc())
		clone.tag.file.Read(0, clonebuff)
		f = file.MakeObservableEditableBuffer("", clonebuff)
	}
	f.AddObserver(&w.tag)
	// w observes tag to update the tag index.
	// TODO(rjk): Add the tag index facility.
	f.AddObserver(w)
	w.tag.file = f

	// Body setup.
	f = file.MakeObservableEditableBuffer("", nil)
	if clone != nil {
		f = clone.body.file
		w.body.org = clone.body.org
	}
	f.AddObserver(&w.body)
	w.body.file = f
	w.filemenu = true
	w.autoindent = *globalAutoIndent
	// w observes body to update the tag in response to actions on the body.
	f.AddTagStatusObserver(w)

	if clone != nil {
		w.autoindent = clone.autoindent
	}
	w.editoutlk = make(chan bool, 1)
	return w
}

func (w *Window) Init(clone *Window, r image.Rectangle, dis draw.Display) {
	w.initHeadless(clone)
	w.display = dis
	r1 := r

	w.tagtop = r
	w.tagtop.Max.Y = r.Min.Y + fontget(global.tagfont, w.display).Height()
	r1.Max.Y = r1.Min.Y + w.taglines*fontget(global.tagfont, w.display).Height()

	w.tag.Init(r1, global.tagfont, global.tagcolors, w.display)
	w.tag.what = Tag

	// When cloning, we copy the tag so that the tag contents can evolve
	// independently.
	if clone != nil {
		w.tag.SetSelect(w.tag.Nc(), w.tag.Nc())
	}
	r1 = r
	r1.Min.Y += w.taglines*fontget(global.tagfont, w.display).Height() + 1
	if r1.Max.Y < r1.Min.Y {
		r1.Max.Y = r1.Min.Y
	}

	var rf string
	if clone != nil {
		rf = clone.body.font
	} else {
		rf = global.tagfont
	}
	w.body.Init(r1, rf, global.textcolors, w.display)
	w.body.what = Body
	r1.Min.Y--
	r1.Max.Y = r1.Min.Y + 1
	if w.display != nil {
		w.display.ScreenImage().Draw(r1, global.tagcolors[frame.ColBord], nil, image.Point{})
	}
	w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
	w.r = r
	var br image.Rectangle
	br.Min = w.tag.scrollr.Min
	br.Max.X = br.Min.X + global.button.R().Dx()
	br.Max.Y = br.Min.Y + global.button.R().Dy()
	if w.display != nil {
		w.display.ScreenImage().Draw(br, global.button, nil, global.button.R().Min)
	}
	w.maxlines = w.body.fr.GetFrameFillStatus().Maxlines
	if clone != nil {
		w.body.SetSelect(clone.body.q0, clone.body.q1)
	}
}

// Display returns the window's display as a ui.MouseMover.
// This implements the ui.MouseWindow interface.
func (w *Window) Display() ui.MouseMover {
	return w.display
}

func (w *Window) DrawButton() {
	b := global.button
	if w.body.file.SaveableAndDirty() {
		b = global.modbutton
	}
	var br image.Rectangle

	br.Min = w.tag.scrollr.Min
	br.Max.X = br.Min.X + b.R().Dx()
	br.Max.Y = br.Min.Y + b.R().Dy()
	if w.display != nil {
		w.display.ScreenImage().Draw(br, b, nil, b.R().Min)
	}
}

func (w *Window) delRunePos() int {
	i := w.tagfilenameend + 2
	if i >= w.tag.Nc() {
		return -1
	}
	return i
}

func (w *Window) moveToDel() {
	n := w.delRunePos()
	if n < 0 {
		return
	}
	if w.display != nil {
		w.display.MoveTo(w.tag.fr.Ptofchar(n).Add(image.Pt(4, w.tag.fr.DefaultFontHeight()-4)))
	}
}

// TagLines computes the number of lines in the tag that can fit in r.
func (w *Window) TagLines(r image.Rectangle) int {
	if !w.tagexpand && !w.showdel {
		return 1
	}
	w.showdel = false
	w.tag.Resize(r, true, true /* noredraw */)
	w.tagsafe = false

	if !w.tagexpand {
		// use just as many lines as needed to show the Del
		n := w.delRunePos()
		if n < 0 {
			return 1
		}
		p := w.tag.fr.Ptofchar(n).Sub(w.tag.fr.Rect().Min)
		return 1 + p.Y/w.tag.fr.DefaultFontHeight()
	}

	// can't use more than we have
	if w.tag.fr.GetFrameFillStatus().Nlines >= w.tag.fr.GetFrameFillStatus().Maxlines {
		return w.tag.fr.GetFrameFillStatus().Maxlines
	}

	// if tag ends with \n, include empty line at end for typing
	n := w.tag.fr.GetFrameFillStatus().Nlines
	if w.tag.file.Nr() > 0 {
		c := w.tag.file.ReadC(w.tag.file.Nr() - 1)
		if c == '\n' {
			n++
		}
	}
	if n == 0 {
		n = 1
	}
	return n
}

// Resize the specified Window to rectangle r.
// TODO(rjk): when collapsing the tag, this is called twice. Once would seem
// sufficient.
// TODO(rjk): This function does not appear to update the Window's rect correctly
// in all cases.
func (w *Window) Resize(r image.Rectangle, safe, keepextra bool) int {
	// log.Printf("Window.Resize r=%v safe=%v keepextra=%v\n", r, safe, keepextra)
	// defer log.Println("Window.Resize End\n")

	// TODO(rjk): Do not leak global event state into this function.
	mouseintag := global.mouse.Point.In(w.tag.all)
	mouseinbody := global.mouse.Point.In(w.body.all)

	// Tagtop is a rectangle corresponding to one line of tag.
	w.tagtop = r
	w.tagtop.Max.Y = r.Min.Y + fontget(global.tagfont, w.display).Height()

	r1 := r
	r1.Max.Y = util.Min(r.Max.Y, r1.Min.Y+w.taglines*fontget(global.tagfont, w.display).Height())

	// If needed, recompute number of lines in tag.
	if !safe || !w.tagsafe || !w.tag.all.Eq(r1) {
		w.taglines = w.TagLines(r)
		r1.Max.Y = util.Min(r.Max.Y, r1.Min.Y+w.taglines*fontget(global.tagfont, w.display).Height())
	}

	// Resize/redraw tag TODO(flux)
	y := r1.Max.Y
	if !safe || !w.tagsafe || !w.tag.all.Eq(r1) {
		w.tag.Resize(r1, true, false /* noredraw */)
		y = w.tag.fr.Rect().Max.Y
		w.DrawButton()
		w.tagsafe = true

		// If mouse is in tag, pull up as tag closes.
		if mouseintag && !global.mouse.Point.In(w.tag.all) {
			p := global.mouse.Point
			p.Y = w.tag.all.Max.Y - 3
			if w.display != nil {
				w.display.MoveTo(p)
			}
		}
		// If mouse is in body, push down as tag expands.
		if mouseinbody && global.mouse.Point.In(w.tag.all) {
			p := global.mouse.Point
			p.Y = w.tag.all.Max.Y + 3
			if w.display != nil {
				w.display.MoveTo(p)
			}
		}
	}
	// Redraw body
	r1 = r
	r1.Min.Y = y
	if !safe || !w.body.all.Eq(r1) {
		oy := y
		if y+1+w.body.fr.DefaultFontHeight() <= r.Max.Y { // room for one line
			r1.Min.Y = y
			r1.Max.Y = y + 1
			if w.display != nil {
				w.display.ScreenImage().Draw(r1, global.tagcolors[frame.ColBord], nil, image.Point{})
			}
			y++
			r1.Min.Y = util.Min(y, r.Max.Y)
			r1.Max.Y = r.Max.Y
		} else {
			r1.Min.Y = y
			r1.Max.Y = y
		}
		// Always resize body Text to maintain canonical rectangle
		// Pass noredraw=true if in preview mode (we'll render ourselves)
		y = w.body.Resize(r1, keepextra, w.previewMode /* noredraw */)
		w.r = r
		w.r.Max.Y = y
		w.body.all.Min.Y = oy

		// Render the appropriate view
		if w.previewMode && w.richBody != nil {
			w.richBody.Render(w.body.all)
		} else {
			w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
		}
	}
	w.maxlines = util.Min(w.body.fr.GetFrameFillStatus().Nlines, util.Max(w.maxlines, w.body.fr.GetFrameFillStatus().Maxlines))
	// TODO(rjk): this value doesn't make sense when we've collapsed
	// the tag if the rectangle update block is not executed.
	return w.r.Max.Y
}

// Lock1 locks just this Window. This is a helper for Lock.
// TODO(rjk): This should be an internal detail of Window.
func (w *Window) lock1(owner int) {
	w.lk.Lock()
	w.ref.Inc()
	w.owner = owner
}

// Lock locks every text/clone of w
func (w *Window) Lock(owner int) {
	w.lk.Lock()
	w.ref.Inc()
	w.owner = owner
	f := w.body.file
	f.AllObservers(func(i interface{}) {
		if t, ok := i.(*Text); ok && t.w != w {
			t.w.lock1(owner)
		}
	})
}

// unlock1 unlocks a single window.
func (w *Window) unlock1() {
	w.owner = 0
	w.Close()
	w.lk.Unlock()
}

// Unlock releases the lock on each clone of w
func (w *Window) Unlock() {
	w.body.file.AllObservers(func(i interface{}) {
		if t, ok := i.(*Text); ok && t.w != w {
			t.w.unlock1()
		}
	})
	w.unlock1()
}

func (w *Window) MouseBut() {
	if w.display != nil {
		w.display.MoveTo(w.tag.scrollr.Min.Add(
			image.Pt(w.tag.scrollr.Dx(), fontget(global.tagfont, w.display).Height()).Div(2)))
	}
}

func (w *Window) Close() {
	if w.ref.Dec() == 0 {
		// Cancel pending preview update timer to prevent callback
		// firing on a closed window.
		if w.previewUpdateTimer != nil {
			w.previewUpdateTimer.Stop()
			w.previewUpdateTimer = nil
		}
		w.previewMode = false
		w.richBody = nil
		xfidlog(w, "del")
		w.tag.file.DelObserver(w)
		w.body.file.DelTagStatusObserver(w)
		w.tag.Close()
		w.body.Close()
		if global.activewin == w {
			global.activewin = nil
		}
	}
}

func (w *Window) Delete() {
	x := w.eventx
	if x != nil {
		w.events = w.events[0:0]
		w.eventx = nil
		x.c <- nil // wake him up
	}
}

func (w *Window) Undo(isundo bool) {
	w.utflastqid = -1
	body := &w.body
	if q0, q1, ok := body.file.Undo(isundo); ok {
		body.q0, body.q1 = q0, q1
	}

	// TODO(rjk): Updates the scrollbar and selection.
	// Be sure not to do this inside of the Undo operation's callbacks.
	body.Show(body.q0, body.q1, true)

	// Undo/Redo bypasses the buffer's Insert/Delete observers, so the
	// preview is not notified through the normal SchedulePreviewUpdate path.
	// Update it immediately — undo is a discrete action, not rapid typing.
	if w.IsPreviewMode() {
		w.UpdatePreview()
	}
}

func (w *Window) SetName(name string) {
	t := &w.body
	t.file.SetName(name)
}

func (w *Window) Type(t *Text, r rune) {
	// In preview mode, route body key events through HandlePreviewKey
	if t.what == Body && w.IsPreviewMode() {
		if w.HandlePreviewKey(r) {
			return
		}
		w.HandlePreviewType(t, r)
		return
	}
	t.Type(r)
}

// TODO(rjk): In the future of File's replacement with undo buffer,
// this method could be renamed to something like "UpdateTag"?
func (w *Window) Commit(t *Text) {
	t.Commit()
	if t.what == Body {
		return
	}
	// TODO(rjk): By virtue of being an observer, we know when this has
	// changed. No need to extract it here unless its changed.
	if w.tagfilenamechanged {
		filename := w.ParseTag()
		if filename != w.body.file.Name() {
			global.seq++
			w.body.file.Mark(global.seq)
			w.SetName(filename)
		}
		w.tagfilenamechanged = false
	}
}

func isDir(r string) (bool, error) {
	f, err := os.Open(r)
	if err != nil {
		return false, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return false, err
	}

	if fi.IsDir() {
		return true, nil
	}

	return false, nil
}

// Should include file lookup be built-in? Or provided by a helper?
// TODO(rjk): This should be provided by an external helper.
func (w *Window) AddIncl(r string) {
	// Tries to open absolute paths, and if fails, tries
	// to use dirname instead.
	d, err := isDir(r)
	if !d {
		if filepath.IsAbs(r) {
			warning(nil, "%s: Not a directory: %v", r, err)
			return
		}
		r = w.body.DirName(r)
		d, err := isDir(r)
		if !d {
			warning(nil, "%s: Not a directory: %v", r, err)
			return
		}
	}
	w.incl = append(w.incl, r)
}

// Clean returns true iff w can be treated as unmodified.
// This will modify the File so that the next call to Clean will return true
// even if this one returned false.
func (w *Window) Clean(conservative bool) bool {
	if w.body.file.IsDirOrScratch() { // don't whine if it's a guide file, error window, etc.
		return true
	}
	if !conservative && w.nopen[QWevent] > 0 {
		return true
	}
	if w.body.file.TreatAsDirty() {
		if w.body.file.Name() != "" {
			warning(nil, "%v modified\n", w.body.file.Name())
		} else {
			if w.body.Nc() < 100 { // don't whine if it's too small
				return true
			}
			warning(nil, "unnamed file modified\n")
		}
		// This toggle permits checking if we can safely destroy the window.
		w.body.file.TreatAsClean()
		return false
	}
	return true
}

// CtlPrint generates the contents of the fsys's acme/<id>/ctl pseduo-file if fonts is true.
// Otherwise, it emits a portion of the per-window dump file contents.
func (w *Window) CtlPrint(fonts bool) string {
	isdir := 0
	if w.body.file.IsDir() {
		isdir = 1
	}
	dirty := 0
	if w.body.file.Dirty() {
		dirty = 1
	}
	buf := fmt.Sprintf("%11d %11d %11d %11d %11d ", w.id, w.tag.Nc(),
		w.body.Nc(), isdir, dirty)
	if fonts {
		// fsys exposes the actual physical font name.
		buf = fmt.Sprintf("%s%11d %s %11d ", buf, w.body.fr.Rect().Dx(),
			quote(fontget(w.body.font, w.display).Name()), w.body.fr.GetMaxtab())
	}
	return buf
}

func (w *Window) Eventf(format string, args ...interface{}) {
	var (
		x *Xfid
	)
	if w.nopen[QWevent] == 0 {
		return
	}
	if w.owner == 0 {
		util.AcmeError("no window owner", nil)
	}
	buffy := new(bytes.Buffer)
	fmt.Fprintf(buffy, format, args...)
	b := buffy.Bytes()

	// TODO(rjk): events should be a bytes.Buffer?

	w.events = append(w.events, byte(w.owner))
	w.events = append(w.events, b...)
	x = w.eventx
	if x != nil {
		w.eventx = nil
		x.c <- nil
	}
}

// ClampAddr clamps address range based on the body buffer.
func (w *Window) ClampAddr() {
	if w.addr.q0 < 0 {
		w.addr.q0 = 0
	}
	if w.addr.q1 < 0 {
		w.addr.q1 = 0
	}
	if w.addr.q0 > w.body.Nc() {
		w.addr.q0 = w.body.Nc()
	}
	if w.addr.q1 > w.body.Nc() {
		w.addr.q1 = w.body.Nc()
	}
}

func (w *Window) UpdateTag(newtagstatus file.TagStatus) {
	// log.Printf("Window.UpdateTag, status %+v, %d", newtagstatus, global.seq)
	w.setTag1()
}

// SelectionContentType identifies the markdown formatting type of a selection.
type SelectionContentType int

const (
	ContentPlain      SelectionContentType = iota // Plain unformatted text
	ContentHeading                                // Heading (# ... ######)
	ContentBold                                   // Bold (**text**)
	ContentItalic                                 // Italic (*text*)
	ContentBoldItalic                             // Bold+Italic (***text***)
	ContentCode                                   // Inline code (`text`)
	ContentCodeBlock                              // Fenced code block (```...```)
	ContentLink                                   // Link ([text](url))
	ContentImage                                  // Image (![alt](url))
	ContentMixed                                  // Selection spans multiple formatting types
)

// SelectionContext holds metadata about the current selection in preview mode,
// including source and rendered positions, content type, and formatting style.
// This is used for context-aware paste operations that adapt formatting based
// on the source and destination context.
type SelectionContext struct {
	SourceStart   int                  // Start offset in source markdown text
	SourceEnd     int                  // End offset in source markdown text
	RenderedStart int                  // Start offset in rendered text
	RenderedEnd   int                  // End offset in rendered text
	ContentType   SelectionContentType // Type of content selected
	PrimaryStyle  rich.Style           // Dominant style of the selection
	CodeLanguage  string               // Language tag for code blocks (e.g., "go")

	IncludesOpenMarker  bool // Selection includes the opening formatting marker
	IncludesCloseMarker bool // Selection includes the closing formatting marker
}

// classifyStyle maps a rich.Style to its SelectionContentType.
func classifyStyle(s rich.Style) SelectionContentType {
	switch {
	case s.Image:
		return ContentImage
	case s.Link:
		return ContentLink
	case s.Code && s.Block:
		return ContentCodeBlock
	case s.Code:
		return ContentCode
	case s.Bold && s.Scale > 1.0:
		return ContentHeading
	case s.Bold && s.Italic:
		return ContentBoldItalic
	case s.Bold:
		return ContentBold
	case s.Italic:
		return ContentItalic
	default:
		return ContentPlain
	}
}

// analyzeSelectionContent examines the spans in the rendered RichText content
// within the given rendered-position range [rStart, rEnd) and determines the
// SelectionContentType. This is used during selection context updates to
// classify what kind of markdown content the user has selected.
func (w *Window) analyzeSelectionContent(rStart, rEnd int) SelectionContentType {
	if w.richBody == nil || rStart >= rEnd {
		return ContentPlain
	}

	content := w.richBody.Content()
	if len(content) == 0 {
		return ContentPlain
	}

	var foundType SelectionContentType
	found := false
	pos := 0

	for _, span := range content {
		runeLen := len([]rune(span.Text))
		spanEnd := pos + runeLen

		// Check if this span overlaps [rStart, rEnd)
		if spanEnd > rStart && pos < rEnd {
			ct := classifyStyle(span.Style)
			if !found {
				foundType = ct
				found = true
			} else if ct != foundType {
				return ContentMixed
			}
		}

		pos = spanEnd
		if pos >= rEnd {
			break
		}
	}

	if !found {
		return ContentPlain
	}
	return foundType
}

// updateSelectionContext reads the current selection from richBody, translates
// the rendered positions to source positions via the previewSourceMap, analyzes
// the content type, and stores the result in w.selectionContext. This should be
// called after each selection change in preview mode.
func (w *Window) updateSelectionContext() {
	if !w.previewMode || w.richBody == nil || w.previewSourceMap == nil {
		w.selectionContext = nil
		return
	}

	p0, p1 := w.richBody.Selection()
	contentType := w.analyzeSelectionContent(p0, p1)

	// Translate rendered positions to source positions.
	srcStart, srcEnd := w.previewSourceMap.ToSource(p0, p1)

	// Determine the primary style from the first overlapping span.
	var primaryStyle rich.Style
	content := w.richBody.Content()
	pos := 0
	for _, span := range content {
		runeLen := len([]rune(span.Text))
		spanEnd := pos + runeLen
		if spanEnd > p0 && pos < p1 {
			primaryStyle = span.Style
			break
		}
		pos = spanEnd
	}

	w.selectionContext = &SelectionContext{
		SourceStart:   srcStart,
		SourceEnd:     srcEnd,
		RenderedStart: p0,
		RenderedEnd:   p1,
		ContentType:   contentType,
		PrimaryStyle:  primaryStyle,
	}
}

// transformForPaste adapts the pasted text based on source and destination
// context. It applies formatting rules: re-wraps formatted text for plain
// destinations, strips markers when destination already has the same format,
// and handles structural elements (headings, code blocks) based on whether
// the text includes a trailing newline.
func transformForPaste(text []byte, sourceCtx, destCtx *SelectionContext) []byte {
	// Pass through when context is missing or text is empty.
	if sourceCtx == nil || destCtx == nil || len(text) == 0 {
		return text
	}

	srcType := sourceCtx.ContentType
	dstType := destCtx.ContentType

	// If source and destination are the same formatting type, strip markers
	// (the destination context already provides the formatting).
	if srcType == dstType {
		return stripMarkers(text, srcType)
	}

	// Handle structural elements (headings, code blocks).
	switch srcType {
	case ContentHeading:
		// Trailing newline means structural paste — preserve as-is.
		if len(text) > 0 && text[len(text)-1] == '\n' {
			return text
		}
		// No trailing newline — strip the heading prefix, treat as text.
		return stripHeadingPrefix(text)

	case ContentCodeBlock:
		// Trailing newline means structural paste — preserve fences.
		if len(text) > 0 && text[len(text)-1] == '\n' {
			return text
		}
		// No trailing newline — just the code text, no fences.
		return text

	case ContentBold:
		if dstType == ContentPlain {
			return wrapWith(text, "**")
		}
		return text

	case ContentItalic:
		if dstType == ContentPlain {
			return wrapWith(text, "*")
		}
		return text

	case ContentBoldItalic:
		if dstType == ContentPlain {
			return wrapWith(text, "***")
		}
		return text

	case ContentCode:
		if dstType == ContentPlain {
			return wrapWith(text, "`")
		}
		return text
	}

	// Plain text or unrecognized — pass through.
	return text
}

// stripMarkers removes formatting markers for same-type paste (e.g., heading prefix).
func stripMarkers(text []byte, ct SelectionContentType) []byte {
	switch ct {
	case ContentHeading:
		return stripHeadingPrefix(text)
	default:
		return text
	}
}

// stripHeadingPrefix removes leading # characters and the following space.
func stripHeadingPrefix(text []byte) []byte {
	i := 0
	for i < len(text) && text[i] == '#' {
		i++
	}
	if i > 0 && i < len(text) && text[i] == ' ' {
		i++
	}
	return text[i:]
}

// wrapWith wraps text in the given marker string (e.g., "**", "*", "`").
func wrapWith(text []byte, marker string) []byte {
	m := []byte(marker)
	result := make([]byte, 0, len(m)+len(text)+len(m))
	result = append(result, m...)
	result = append(result, text...)
	result = append(result, m...)
	return result
}

// IsPreviewMode returns true if the window is in preview mode (showing rendered markdown).
func (w *Window) IsPreviewMode() bool {
	return w.previewMode
}

// SetPreviewMode enables or disables preview mode.
// When disabling preview mode, triggers a full redraw of the body.
// The image cache is kept alive so re-entering preview is fast.
func (w *Window) SetPreviewMode(enabled bool) {
	wasPreview := w.previewMode
	w.previewMode = enabled

	// When exiting preview mode, refresh the body.
	// Keep the image cache alive so re-entering preview is fast.
	if wasPreview && !enabled {
		// Force a full redraw of the body by resizing it
		if w.display != nil {
			w.body.Resize(w.body.all, true, false)
			w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
			w.display.Flush()
		}
	}
}

// TogglePreviewMode toggles the preview mode state.
func (w *Window) TogglePreviewMode() {
	w.SetPreviewMode(!w.previewMode)
}

// RichBody returns the rich text renderer for preview mode, or nil if not initialized.
func (w *Window) RichBody() *RichText {
	return w.richBody
}

// Draw renders the window. In preview mode, it renders the richBody;
// otherwise, it uses the normal body rendering.
func (w *Window) Draw() {
	if w.previewMode && w.richBody != nil {
		w.richBody.Render(w.body.all)
	} else {
		// Normal body rendering is handled by the existing Text.Redraw
		// mechanism which is called through Text.Resize and other paths.
		// For explicit Draw() calls, we trigger a redraw of the body frame.
		if w.body.fr != nil {
			enclosing := w.body.fr.Rect()
			if w.display != nil {
				enclosing.Min.X -= w.display.ScaleSize(Scrollwid + Scrollgap)
			}
			w.body.fr.Redraw(enclosing)
		}
	}
}

// HandlePreviewMouse handles mouse events when the window is in preview mode.
// Returns true if the event was handled by the preview mode, false otherwise.
// When false is returned, the caller should handle the event normally.
func (w *Window) HandlePreviewMouse(m *draw.Mouse, mc *draw.Mousectl) bool {
	if !w.previewMode || w.richBody == nil {
		return false
	}

	// Check if the mouse is in the body area
	if !m.Point.In(w.body.all) {
		return false
	}

	rt := w.richBody

	// Handle scroll wheel (buttons 4 and 5).
	// When the cursor is over a horizontally-scrollable block region,
	// redirect vertical scroll to horizontal scrolling.
	if m.Buttons&8 != 0 || m.Buttons&16 != 0 {
		if regionIndex, ok := rt.Frame().PointInBlockRegion(m.Point); ok {
			// Horizontal scroll: button 4 = left, button 5 = right.
			delta := 40 // pixels per scroll tick
			if m.Buttons&8 != 0 {
				delta = -delta
			}
			rt.Frame().HScrollWheel(delta, regionIndex)
		} else {
			// Normal vertical scroll.
			up := m.Buttons&8 != 0
			rt.ScrollWheel(up)
		}
		w.Draw()
		if w.display != nil {
			w.display.Flush()
		}
		return true
	}

	// Handle scrollbar clicks (buttons 1, 2, 3 in scrollbar area).
	// Uses latching: once pressed, the scroll tracks the mouse until release.
	scrRect := rt.ScrollRect()
	if m.Point.In(scrRect) {
		button := 0
		if m.Buttons&1 != 0 {
			button = 1
		} else if m.Buttons&2 != 0 {
			button = 2
		} else if m.Buttons&4 != 0 {
			button = 3
		}
		if button != 0 && mc != nil {
			w.previewVScrollLatch(rt, mc, button, scrRect)
			return true
		}
	}

	// Handle horizontal scrollbar clicks (buttons 1, 2, 3 on h-scrollbar).
	// Uses latching: same pattern as vertical.
	if regionIndex, ok := rt.Frame().HScrollBarAt(m.Point); ok {
		button := 0
		if m.Buttons&1 != 0 {
			button = 1
		} else if m.Buttons&2 != 0 {
			button = 2
		} else if m.Buttons&4 != 0 {
			button = 3
		}
		if button != 0 && mc != nil {
			w.previewHScrollLatch(rt, mc, button, regionIndex)
			return true
		}
	}

	// Handle button 1 in frame area for text selection and chording.
	// Chord processing follows the same pattern as text.go: chords are
	// handled inline in a loop while B1 is held, so that sequential
	// B1+B2 (cut) then B1+B3 (paste) works correctly.
	frameRect := rt.Frame().Rect()
	if m.Point.In(frameRect) && m.Buttons&1 != 0 && mc != nil {
		var p0, p1 int
		var lastButtons int // track button state for chord loop

		selectq := rt.Frame().Charofpt(m.Point)
		b := m.Buttons
		fr := rt.Frame()

		// Check for double-click: same richtext, same position, within 500ms.
		prevP0, prevP1 := rt.Selection()
		if w.previewClickRT == rt &&
			m.Msec-w.previewClickMsec < 500 &&
			prevP0 == prevP1 && selectq == prevP0 {

			// Double-click: expand selection
			p0, p1 = fr.ExpandAtPos(selectq)
			rt.SetSelection(p0, p1)
			fr.Redraw()
			if w.display != nil {
				w.display.Flush()
			}
			w.previewClickRT = nil

			// Wait for mouse state change (jitter tolerance), then
			// fall through to the chord processing loop below.
			x, y := m.Point.X, m.Point.Y
			for {
				me := <-mc.C
				lastButtons = me.Buttons
				if !(me.Buttons == b &&
					util.Abs(me.Point.X-x) < 3 &&
					util.Abs(me.Point.Y-y) < 3) {
					break
				}
			}
		} else {
			// Normal click/drag selection: track drag until a chord
			// is detected or all buttons are released.
			anchor := selectq
			for {
				me := <-mc.C
				lastButtons = me.Buttons
				current := fr.Charofpt(me.Point)
				if anchor <= current {
					p0, p1 = anchor, current
				} else {
					p0, p1 = current, anchor
				}
				rt.SetSelection(p0, p1)
				fr.Redraw()
				if w.display != nil {
					w.display.Flush()
				}
				// Chord detected or all buttons released: exit drag.
				if me.Buttons != b || me.Buttons == 0 {
					break
				}
			}

			// Record double-click state
			if p0 == p1 {
				w.previewClickRT = rt
				w.previewClickPos = p0
				w.previewClickMsec = m.Msec
			} else {
				w.previewClickRT = nil
			}
		}

		// Sync the preview selection to the source body buffer
		w.syncSourceSelection()
		q0 := w.body.q0

		// Chord processing loop: handle B2/B3 chords while B1 is held,
		// matching the text.go pattern with undo/redo toggle semantics.
		const (
			chordNone = iota
			chordCut
			chordPaste
			chordSnarf
		)
		state := chordNone
		for lastButtons != 0 {
			if lastButtons == 7 && state == chordNone {
				// B1+B2+B3 simultaneous: snarf only (copy, no delete)
				cut(&w.body, &w.body, nil, true, false, "")
				global.snarfContext = w.selectionContext
				state = chordSnarf
			} else if (lastButtons&1) != 0 && (lastButtons&6) != 0 && state != chordSnarf {
				if state == chordNone {
					w.body.TypeCommit()
					global.seq++
					w.body.file.Mark(global.seq)
				}
				if lastButtons&2 != 0 {
					// B2 chord: cut (or undo a previous paste)
					if state == chordPaste {
						w.Undo(true)
						w.body.SetSelect(q0, w.body.q1)
						state = chordNone
					} else if state != chordCut {
						cut(&w.body, &w.body, nil, true, true, "")
						global.snarfContext = w.selectionContext
						state = chordCut
					}
				} else {
					// B3 chord: paste (or undo a previous cut)
					if state == chordCut {
						w.Undo(true)
						w.body.SetSelect(q0, w.body.q1)
						state = chordNone
					} else if state != chordPaste {
						paste(&w.body, &w.body, nil, true, false, "")
						state = chordPaste
					}
				}
				// Collapse the rich frame's selection before re-rendering
				// so UpdatePreview doesn't draw a stale highlight.
				mq0, mq1 := w.previewSourceMap.ToRendered(w.body.q0, w.body.q1)
				rt.SetSelection(mq0, mq1)
				w.UpdatePreview()
				// Now use the new source map to set the correct selection.
				if w.previewSourceMap != nil {
					rendStart, rendEnd := w.previewSourceMap.ToRendered(w.body.q0, w.body.q1)
					if rendStart >= 0 {
						rt.SetSelection(rendStart, rendEnd)
					}
				}
				clearmouse()
			}
			// Wait for button state to change
			prev := lastButtons
			for lastButtons == prev {
				me := <-mc.C
				lastButtons = me.Buttons
			}
			w.previewClickRT = nil
		}

		w.Draw()
		if w.display != nil {
			w.display.Flush()
		}
		return true
	}

	// Handle button 2 (B2/middle-click) in frame area for Execute action
	if m.Point.In(frameRect) && m.Buttons&2 != 0 {
		// Save the prior selection to restore after B2 execute
		priorP0, priorP1 := rt.Selection()

		var p0, p1 int
		if mc != nil {
			// Use Frame.SelectWithColor() for proper drag selection with B2
			// Pass global.but2col (red) for colored sweep during drag
			p0, p1 = rt.Frame().SelectWithColor(mc, m, global.but2col)
			rt.SetSelection(p0, p1)
		} else {
			// Fallback: just set point selection if no Mousectl available
			charPos := rt.Frame().Charofpt(m.Point)
			p0, p1 = charPos, charPos
			rt.SetSelection(charPos, charPos)
		}
		// If null click (no sweep), expand selection. In a code block,
		// expands to the full block; otherwise expands to word.
		if p0 == p1 {
			q0, q1 := rt.Frame().ExpandAtPos(p0)
			if q0 != q1 {
				rt.SetSelection(q0, q1)
				p0, p1 = q0, q1
			}
		}
		// Sync the preview selection to the source body buffer
		w.syncSourceSelection()
		w.Draw()
		if w.display != nil {
			w.display.Flush()
		}
		// Execute the rendered text as a command
		cmdText := w.PreviewExecText()
		if cmdText != "" {
			previewExecute(&w.body, cmdText)
		}
		// Restore prior selection after B2 execute action
		rt.SetSelection(priorP0, priorP1)
		w.Draw()
		if w.display != nil {
			w.display.Flush()
		}
		return true
	}

	// Handle button 3 (B3/right-click) in frame area for Look action
	if m.Point.In(frameRect) && m.Buttons&4 != 0 {
		// First, perform sweep selection (like B1/B2)
		var p0, p1 int
		if mc != nil {
			// Use Frame.SelectWithColor() for proper drag selection with B3
			// Pass global.but3col (green) for colored sweep during drag
			p0, p1 = rt.Frame().SelectWithColor(mc, m, global.but3col)
			rt.SetSelection(p0, p1)
		} else {
			charPos := rt.Frame().Charofpt(m.Point)
			p0, p1 = charPos, charPos
			rt.SetSelection(charPos, charPos)
		}

		// Determine the character position for link/image checks
		charPos := p0

		// If null click (no sweep), check for existing selection or expand.
		// In a code block, expands to the full block; otherwise expands to word.
		if p0 == p1 {
			// Check if click is inside an existing selection
			q0, q1 := rt.Selection()
			if q0 != q1 && charPos >= q0 && charPos < q1 {
				// Click inside existing selection - use it as-is
				p0, p1 = q0, q1
				rt.SetSelection(q0, q1)
			} else {
				q0, q1 := rt.Frame().ExpandAtPos(p0)
				if q0 != q1 {
					rt.SetSelection(q0, q1)
					p0, p1 = q0, q1
				}
			}
		}

		// Check if this position is within a link
		url := w.PreviewLookLinkURL(charPos)

		if url != "" {
			// Plumb the URL using the same mechanism as look3
			if plumbsendfid != nil {
				pm := &plumb.Message{
					Src:  "acme",
					Dst:  "",
					Dir:  w.body.AbsDirName(""),
					Type: "text",
					Data: []byte(url),
				}
				if err := pm.Send(plumbsendfid); err != nil {
					warning(nil, "Markdeep B3: plumb failed: %v\n", err)
				}
			} else {
				warning(nil, "Markdeep B3: plumber not running\n")
			}
			return true
		}

		// Check if this position is within an image
		imageURL := rt.Frame().ImageURLAt(charPos)
		if imageURL != "" {
			// Plumb the image path
			if plumbsendfid != nil {
				pm := &plumb.Message{
					Src:  "acme",
					Dst:  "",
					Dir:  w.body.AbsDirName(""),
					Type: "text",
					Data: []byte(imageURL),
				}
				if err := pm.Send(plumbsendfid); err != nil {
					warning(nil, "Markdeep B3 image: plumb failed: %v\n", err)
				}
			} else {
				warning(nil, "Markdeep B3 image: plumber not running\n")
			}
			return true
		}

		// Not a link or image - use selected text for Look (search in body)
		w.syncSourceSelection()

		// Read source text from body selection (not rendered text)
		// This ensures we search for the actual markdown source, not stripped text
		srcLen := w.body.q1 - w.body.q0
		if srcLen > 0 {
			srcBuf := make([]rune, srcLen)
			w.body.file.Read(w.body.q0, srcBuf)

			// Search source buffer for the source text
			if search(&w.body, srcBuf) {
				// Map the search result (body.q0/q1) back to rendered positions
				if w.previewSourceMap != nil {
					rendStart, rendEnd := w.previewSourceMap.ToRendered(w.body.q0, w.body.q1)
					if rendStart >= 0 && rendEnd >= 0 {
						rt.SetSelection(rendStart, rendEnd)
						w.scrollPreviewToMatch(rt, rendStart)
						// Warp cursor to found text, matching normal Acme's look3() behavior
						if w.display != nil {
							warpPt := rt.Frame().Ptofchar(rendStart).Add(
								image.Pt(4, rt.Frame().DefaultFontHeight()-4))
							w.display.MoveTo(warpPt)
						}
					}
				}
			}
		}

		w.Draw()
		if w.display != nil {
			w.display.Flush()
		}
		return true
	}

	return false
}

// previewScrSleep waits for dt milliseconds or until a mouse event arrives,
// whichever comes first. This matches ScrSleep in scrl.go but reads from the
// passed-in Mousectl rather than global.mousectl.
func previewScrSleep(mc *draw.Mousectl, dt int) {
	timer := time.NewTimer(time.Duration(dt) * time.Millisecond)
	select {
	case <-timer.C:
	case mc.Mouse = <-mc.C:
		timer.Stop()
	}
}

// previewVScrollLatch implements acme-style latching for the vertical
// scrollbar in preview mode. Once a button is pressed in the scrollbar, the
// scroll action tracks the mouse until the button is released. The cursor is
// physically warped back into the scrollbar on each iteration, matching the
// acme pattern in scrl.go.
func (w *Window) previewVScrollLatch(rt *RichText, mc *draw.Mousectl, button int, scrRect image.Rectangle) {
	buttonBit := 1 << uint(button-1)
	centerX := (scrRect.Min.X + scrRect.Max.X) / 2

	// Initial scroll action.
	rt.ScrollClick(button, mc.Mouse.Point)
	w.Draw()
	if w.display != nil {
		w.display.Flush()
	}

	first := true
	for {
		if button == 2 {
			// B2: read per-event for live thumb drag.
			mc.Mouse = <-mc.C
		} else {
			// B1/B3: debounce for auto-repeat.
			if first {
				if w.display != nil {
					w.display.Flush()
				}
				time.Sleep(200 * time.Millisecond)
				mc.Mouse = <-mc.C
				first = false
			} else {
				previewScrSleep(mc, 80)
			}
		}

		if mc.Mouse.Buttons&buttonBit == 0 {
			break
		}

		// Clamp Y and lock X to center of scrollbar.
		my := mc.Mouse.Point.Y
		if my < scrRect.Min.Y {
			my = scrRect.Min.Y
		}
		if my >= scrRect.Max.Y {
			my = scrRect.Max.Y
		}
		warpPt := image.Pt(centerX, my)

		// Warp cursor back into scrollbar if it has moved away.
		if !mc.Mouse.Point.Eq(warpPt) {
			if w.display != nil {
				w.display.MoveTo(warpPt)
			}
			mc.Mouse = <-mc.C // absorb synthetic move event
		}

		rt.ScrollClick(button, warpPt)
		w.Draw()
		if w.display != nil {
			w.display.Flush()
		}
	}

	// Drain remaining mouse events until all buttons released.
	for mc.Mouse.Buttons != 0 {
		mc.Mouse = <-mc.C
	}
}

// previewHScrollLatch implements acme-style latching for horizontal
// scrollbars in preview mode. Same pattern as previewVScrollLatch but for the
// horizontal axis. The cursor is warped to stay within the scrollbar band.
func (w *Window) previewHScrollLatch(rt *RichText, mc *draw.Mousectl, button int, regionIndex int) {
	buttonBit := 1 << uint(button-1)
	frameRect := rt.Frame().Rect()

	// Get the scrollbar rectangle for cursor warping.
	barRect := rt.Frame().HScrollBarRect(regionIndex)
	centerY := (barRect.Min.Y + barRect.Max.Y) / 2

	// Initial scroll action.
	rt.Frame().HScrollClick(button, mc.Mouse.Point, regionIndex)
	w.Draw()
	if w.display != nil {
		w.display.Flush()
	}

	first := true
	for {
		if button == 2 {
			mc.Mouse = <-mc.C
		} else {
			if first {
				if w.display != nil {
					w.display.Flush()
				}
				time.Sleep(200 * time.Millisecond)
				mc.Mouse = <-mc.C
				first = false
			} else {
				previewScrSleep(mc, 80)
			}
		}

		if mc.Mouse.Buttons&buttonBit == 0 {
			break
		}

		// Clamp X to scrollbar bounds and lock Y to center of scrollbar band.
		mx := mc.Mouse.Point.X
		if mx < barRect.Min.X {
			mx = barRect.Min.X
		}
		if mx >= frameRect.Max.X {
			mx = frameRect.Max.X
		}
		warpPt := image.Pt(mx, centerY)

		// Warp cursor back into scrollbar if it has moved away.
		if !mc.Mouse.Point.Eq(warpPt) {
			if w.display != nil {
				w.display.MoveTo(warpPt)
			}
			mc.Mouse = <-mc.C // absorb synthetic move event
		}

		rt.Frame().HScrollClick(button, warpPt, regionIndex)
		w.Draw()
		if w.display != nil {
			w.display.Flush()
		}
	}

	for mc.Mouse.Buttons != 0 {
		mc.Mouse = <-mc.C
	}
}

// ShowInPreview maps source positions [q0, q1) to rendered positions,
// updates the preview selection, scrolls to make it visible, and redraws.
// Returns the rendered start position (for cursor warping), or -1 if
// mapping failed.
func (w *Window) ShowInPreview(q0, q1 int) int {
	if !w.previewMode || w.richBody == nil || w.previewSourceMap == nil {
		return -1
	}
	rt := w.richBody
	rendStart, rendEnd := w.previewSourceMap.ToRendered(q0, q1)
	if rendStart < 0 || rendEnd < 0 {
		return -1
	}
	rt.SetSelection(rendStart, rendEnd)
	w.scrollPreviewToMatch(rt, rendStart)
	w.Draw()
	if w.display != nil {
		w.display.Flush()
	}
	return rendStart
}

// scrollPreviewToMatch scrolls the preview so that the match at rendStart
// is visible, placing it roughly 1/3 from the top of the frame (matching
// Acme's Show() scroll behavior). If the match is already visible, no
// scrolling occurs.
func (w *Window) scrollPreviewToMatch(rt *RichText, rendStart int) {
	fr := rt.Frame()
	if fr == nil {
		return
	}

	// Get layout information
	lineStarts := fr.LineStartRunes()
	maxLines := fr.MaxLines()
	if maxLines <= 0 || len(lineStarts) == 0 {
		return
	}

	// Find which line contains rendStart
	origin := rt.Origin()
	matchLine := 0
	for i, start := range lineStarts {
		if rendStart >= start {
			matchLine = i
		} else {
			break
		}
	}

	// Find which line corresponds to the current origin
	originLine := 0
	for i, start := range lineStarts {
		if origin >= start {
			originLine = i
		} else {
			break
		}
	}

	// Check if the match is already visible
	if matchLine >= originLine && matchLine < originLine+maxLines {
		return
	}

	// Scroll so the match is ~1/3 from the top
	targetLine := matchLine - maxLines/3
	if targetLine < 0 {
		targetLine = 0
	}
	if targetLine < len(lineStarts) {
		rt.SetOrigin(lineStarts[targetLine])
	}
}

// SetPreviewSourceMap sets the source map used for mapping rendered positions
// to source positions when in preview mode.
func (w *Window) SetPreviewSourceMap(sm *markdown.SourceMap) {
	w.previewSourceMap = sm
}

// PreviewSourceMap returns the current source map, or nil if not set.
func (w *Window) PreviewSourceMap() *markdown.SourceMap {
	return w.previewSourceMap
}

// SetPreviewLinkMap sets the link map used for mapping rendered positions
// to link URLs when in preview mode.
func (w *Window) SetPreviewLinkMap(lm *markdown.LinkMap) {
	w.previewLinkMap = lm
}

// PreviewLinkMap returns the current link map, or nil if not set.
func (w *Window) PreviewLinkMap() *markdown.LinkMap {
	return w.previewLinkMap
}

// PreviewLookLinkURL returns the URL if the given position in the rendered preview
// falls within a link. Returns empty string if the position is not within a link,
// if not in preview mode, or if no link map is set.
// This is used by the Look handler to determine if a B3 click should open a URL.
func (w *Window) PreviewLookLinkURL(pos int) string {
	if !w.previewMode || w.previewLinkMap == nil {
		return ""
	}
	return w.previewLinkMap.URLAt(pos)
}

// recordEdit accumulates an edit record for the incremental preview path.
func (w *Window) recordEdit(e markdown.EditRecord) {
	w.pendingEdits = append(w.pendingEdits, e)
}

// UpdatePreview updates the preview content from the body buffer.
// This should be called when the body buffer changes and the window is in preview mode.
// It re-parses the markdown and updates the richBody, preserving the scroll position.
// When edit position information is available (from pendingEdits), it attempts an
// incremental re-parse of only the affected blocks. Falls back to full re-parse
// when the incremental path cannot determine the affected region.
func (w *Window) UpdatePreview() {
	if !w.previewMode || w.richBody == nil {
		return
	}

	// Get the current scroll position to preserve it
	currentOrigin := w.richBody.Origin()

	// Read the current body content
	bodyContent := w.body.file.String()

	var content rich.Content
	var sourceMap *markdown.SourceMap
	var linkMap *markdown.LinkMap
	var blockIdx *markdown.BlockIndex

	// Try incremental path when we have a previous block index and pending edits.
	if w.prevBlockIndex != nil && len(w.pendingEdits) > 0 {
		old := markdown.StitchResult{
			Content:  w.richBody.Content(),
			SM:       w.previewSourceMap,
			LM:       w.previewLinkMap,
			BlockIdx: w.prevBlockIndex,
		}
		result, ok := markdown.IncrementalUpdate(old, bodyContent, w.pendingEdits)
		if ok {
			content = result.Content
			sourceMap = result.SM
			linkMap = result.LM
			blockIdx = result.BlockIdx
		}
	}

	// Full re-parse fallback.
	if content == nil {
		content, sourceMap, linkMap, blockIdx = markdown.ParseWithSourceMapAndIndex(bodyContent)
	}

	// Clear pending edits.
	w.pendingEdits = w.pendingEdits[:0]

	// Update the rich body content
	w.richBody.SetContent(content)
	w.previewSourceMap = sourceMap
	w.previewLinkMap = linkMap
	w.prevBlockIndex = blockIdx

	// Try to restore the scroll position
	// Clamp to the new content length if necessary
	newLen := content.Len()
	if currentOrigin > newLen {
		currentOrigin = newLen
	}
	w.richBody.SetOrigin(currentOrigin)

	// Render the preview using body.all as the canonical geometry
	w.richBody.Render(w.body.all)
	if w.display != nil {
		w.display.Flush()
	}
}

// previewUpdateDelay is the duration to wait after the last edit before
// re-rendering the Markdeep preview. This debounces rapid edits.
const previewUpdateDelay = 3 * time.Second

// SchedulePreviewUpdate schedules a debounced update of the Markdeep preview.
// If called multiple times in quick succession, only the last call will trigger
// an update after the delay. This prevents excessive re-rendering during editing.
//
// Must be called with global.row.lk held.
func (w *Window) SchedulePreviewUpdate() {
	if !w.previewMode || w.richBody == nil {
		return
	}

	// Cancel any pending update
	if w.previewUpdateTimer != nil {
		w.previewUpdateTimer.Stop()
	}

	// Schedule a new update after the delay.
	// The callback runs on a timer goroutine; acquire the row lock
	// to synchronize with mousethread, keyboardthread, and others.
	w.previewUpdateTimer = time.AfterFunc(previewUpdateDelay, func() {
		global.row.lk.Lock()
		defer global.row.lk.Unlock()

		// Re-check: window may have exited preview mode or been closed
		// while we waited for the lock.
		if !w.previewMode || w.richBody == nil {
			return
		}

		w.UpdatePreview()
	})
}

// PreviewSnarf returns the text that would be snarfed (copied) when in preview mode.
// It uses the source map to convert the selection in the rendered rich text back to
// positions in the source markdown, then extracts that range from the body buffer.
// Returns empty slice if not in preview mode, no rich body, no selection, or no source map.
func (w *Window) PreviewSnarf() []byte {
	if !w.previewMode || w.richBody == nil || w.previewSourceMap == nil {
		return nil
	}

	// Get selection from the rich text frame
	p0, p1 := w.richBody.Selection()
	if p0 == p1 {
		return nil // No selection
	}

	// Map rendered positions to source positions
	srcStart, srcEnd := w.previewSourceMap.ToSource(p0, p1)

	// Clamp to body buffer bounds
	srcStart, srcEnd = clampToBuffer(srcStart, srcEnd, w.body.file.Nr())
	if srcStart >= srcEnd {
		return nil
	}

	// Read the source text from the body buffer
	buf := make([]rune, srcEnd-srcStart)
	w.body.file.Read(srcStart, buf)

	return []byte(string(buf))
}

// PreviewLookText returns the selected text from the preview for a Look (B3) operation.
// In preview mode, this returns the rendered text (not the source markdown).
// Returns empty string if not in preview mode or no selection.
func (w *Window) PreviewLookText() string {
	if !w.previewMode || w.richBody == nil {
		return ""
	}

	// Get selection from the rich text frame
	p0, p1 := w.richBody.Selection()
	if p0 == p1 {
		return "" // No selection
	}

	// Get the plain text from the rendered content
	content := w.richBody.Content()
	if content == nil {
		return ""
	}

	plainText := content.Plain()
	if p0 < 0 || p1 < 0 || p0 > p1 || p1 > len(plainText) {
		return ""
	}

	return string(plainText[p0:p1])
}

// PreviewExecText returns the selected text from the preview for an Exec (B2) operation.
// In preview mode, this returns the rendered text (not the source markdown).
// Returns empty string if not in preview mode or no selection.
func (w *Window) PreviewExecText() string {
	// Exec and Look use the same text extraction logic
	return w.PreviewLookText()
}

// PreviewExpandWord expands a click position to the full word in preview mode.
// Given a position in the rendered text, returns the word containing that position
// along with its start and end positions. Used for B3 Look when there's no selection.
func (w *Window) PreviewExpandWord(pos int) (word string, start, end int) {
	if !w.previewMode || w.richBody == nil {
		return "", pos, pos
	}

	content := w.richBody.Content()
	if content == nil {
		return "", pos, pos
	}

	plainText := content.Plain()
	n := len(plainText)

	if pos < 0 || pos >= n {
		return "", pos, pos
	}

	// Expand left to find word start
	start = pos
	for start > 0 && isWordChar(plainText[start-1]) {
		start--
	}

	// Expand right to find word end
	end = pos
	for end < n && isWordChar(plainText[end]) {
		end++
	}

	if start >= end {
		return "", pos, pos
	}

	return string(plainText[start:end]), start, end
}

// isWordChar returns true if the rune is part of a word (alphanumeric or underscore).
func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// clampToBuffer clamps start and end positions to [0, bufLen].
// If clamping causes start > end, start is set to end.
func clampToBuffer(start, end, bufLen int) (int, int) {
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > bufLen {
		start = bufLen
	}
	if end > bufLen {
		end = bufLen
	}
	if start > end {
		start = end
	}
	return start, end
}

// syncSourceSelection maps the current preview selection to the corresponding
// positions in the source body buffer. This keeps body.q0 and body.q1 in sync
// with the rendered preview selection, enabling Snarf and other Acme operations
// to work correctly in preview mode.
func (w *Window) syncSourceSelection() {
	if !w.previewMode || w.richBody == nil || w.previewSourceMap == nil {
		return
	}

	// Get selection from the rich text frame
	p0, p1 := w.richBody.Selection()

	// Map rendered positions to source positions
	srcStart, srcEnd := w.previewSourceMap.ToSource(p0, p1)

	// Clamp to body buffer bounds
	srcStart, srcEnd = clampToBuffer(srcStart, srcEnd, w.body.file.Nr())

	// Update the body's selection to match
	w.body.q0 = srcStart
	w.body.q1 = srcEnd
}

// HandlePreviewKey handles keyboard input when the window is in preview mode.
// Returns true if the key was handled (navigation keys), false otherwise (typing keys).
// Navigation keys (Page Up/Down, arrows, Home, End) scroll the preview.
// Escape exits preview mode.
// Typing keys are ignored in preview mode (returns false to indicate not handled).
func (w *Window) HandlePreviewKey(key rune) bool {
	if !w.previewMode || w.richBody == nil {
		return false
	}

	rt := w.richBody
	frame := rt.Frame()
	if frame == nil {
		return false
	}

	// Helper to count lines and get line start positions in the content
	getLineInfo := func() (lineCount int, lineStarts []int) {
		content := rt.Content()
		if content == nil {
			return 1, []int{0}
		}
		lineCount = 1
		lineStarts = []int{0}
		runeOffset := 0
		for _, span := range content {
			for _, r := range span.Text {
				if r == '\n' {
					lineCount++
					lineStarts = append(lineStarts, runeOffset+1)
				}
				runeOffset++
			}
		}
		return lineCount, lineStarts
	}

	// Helper to find origin for a specific line
	findOriginForLine := func(line int, lineStarts []int) int {
		if line < 0 {
			return 0
		}
		if line >= len(lineStarts) {
			return lineStarts[len(lineStarts)-1]
		}
		return lineStarts[line]
	}

	// Helper to find current line from origin
	findCurrentLine := func(origin int, lineStarts []int) int {
		currentLine := 0
		for i, start := range lineStarts {
			if origin >= start {
				currentLine = i
			} else {
				break
			}
		}
		return currentLine
	}

	switch key {
	case draw.KeyPageDown:
		// Scroll down by a page
		lineCount, lineStarts := getLineInfo()
		maxLines := frame.MaxLines()
		if maxLines <= 0 {
			maxLines = 10
		}
		currentLine := findCurrentLine(rt.Origin(), lineStarts)
		newLine := currentLine + maxLines
		if newLine >= lineCount {
			newLine = lineCount - 1
		}
		rt.SetOrigin(findOriginForLine(newLine, lineStarts))
		rt.Redraw()
		return true

	case draw.KeyPageUp:
		// Scroll up by a page
		_, lineStarts := getLineInfo()
		maxLines := frame.MaxLines()
		if maxLines <= 0 {
			maxLines = 10
		}
		currentLine := findCurrentLine(rt.Origin(), lineStarts)
		newLine := currentLine - maxLines
		if newLine < 0 {
			newLine = 0
		}
		rt.SetOrigin(findOriginForLine(newLine, lineStarts))
		rt.Redraw()
		return true

	case draw.KeyDown:
		// Scroll down by one line
		lineCount, lineStarts := getLineInfo()
		currentLine := findCurrentLine(rt.Origin(), lineStarts)
		newLine := currentLine + 1
		if newLine >= lineCount {
			newLine = lineCount - 1
		}
		rt.SetOrigin(findOriginForLine(newLine, lineStarts))
		rt.Redraw()
		return true

	case draw.KeyUp:
		// Scroll up by one line
		_, lineStarts := getLineInfo()
		currentLine := findCurrentLine(rt.Origin(), lineStarts)
		newLine := currentLine - 1
		if newLine < 0 {
			newLine = 0
		}
		rt.SetOrigin(findOriginForLine(newLine, lineStarts))
		rt.Redraw()
		return true

	case draw.KeyHome:
		// Scroll to beginning
		rt.SetOrigin(0)
		rt.Redraw()
		return true

	case draw.KeyEnd:
		// Scroll to end
		lineCount, lineStarts := getLineInfo()
		maxLines := frame.MaxLines()
		if maxLines <= 0 {
			maxLines = 10
		}
		// Position so that last lines are visible
		endLine := lineCount - maxLines
		if endLine < 0 {
			endLine = 0
		}
		rt.SetOrigin(findOriginForLine(endLine, lineStarts))
		rt.Redraw()
		return true

	case 0x1B: // Escape
		// Exit preview mode
		w.SetPreviewMode(false)
		return true

	default:
		// Typing keys and other keys are not handled in preview mode
		return false
	}
}

// HandlePreviewType handles text editing keys in preview mode, inserting or
// deleting characters in the source buffer and immediately re-rendering the
// preview. It follows the same sync→edit→render→remap cycle used by the
// chord cut/paste handlers.
func (w *Window) HandlePreviewType(t *Text, r rune) {
	if !w.previewMode || w.richBody == nil || w.previewSourceMap == nil {
		return
	}

	// Only accept printable characters, newline, tab, and editing keys.
	switch {
	case r == '\n', r == '\t':
		// accepted
	case r == 0x08: // ^H: backspace
	case r == 0x7F: // Del: delete right
	case r == 0x15: // ^U: kill line
	case r == 0x17: // ^W: kill word
	case r >= 0x20 && r < KF: // printable, excluding draw key constants (0xF0xx, 0xF1xx)
		// accepted
	default:
		return
	}

	// 1. Map rendered cursor/selection to source positions.
	w.syncSourceSelection()

	// 2. Create undo points matching text mode behavior.
	// Deletion keys and newline always start a new undo group.
	// Regular characters only create an undo point at the start of
	// a typing sequence (eq0 == -1), so consecutive chars are grouped
	// into one Undo operation.
	switch r {
	case 0x08, 0x7F, 0x15, 0x17, '\n': // deletion keys and newline
		t.TypeCommit()
		global.seq++
		t.file.Mark(global.seq)
	default:
		if t.eq0 == -1 {
			t.TypeCommit()
			global.seq++
			t.file.Mark(global.seq)
		}
	}

	// 3. Handle deletion keys.
	switch r {
	case 0x08: // ^H: backspace
		if t.q0 != t.q1 {
			// Range selected: delete it.
			cut(t, t, nil, false, true, "")
		} else if t.q0 > 0 {
			t.q0--
			cut(t, t, nil, false, true, "")
		}
		w.previewTypeFinish(t)
		return

	case 0x7F: // Del: delete right
		if t.q0 != t.q1 {
			cut(t, t, nil, false, true, "")
		} else if t.q1 < t.file.Nr() {
			t.q1++
			cut(t, t, nil, false, true, "")
		}
		w.previewTypeFinish(t)
		return

	case 0x15: // ^U: kill line
		if t.q0 != t.q1 {
			cut(t, t, nil, false, true, "")
		} else if t.q0 > 0 {
			nnb := t.BsWidth(0x15)
			if nnb > 0 {
				t.q0 -= nnb
				cut(t, t, nil, false, true, "")
			}
		}
		w.previewTypeFinish(t)
		return

	case 0x17: // ^W: kill word
		if t.q0 != t.q1 {
			cut(t, t, nil, false, true, "")
		} else if t.q0 > 0 {
			nnb := t.BsWidth(0x17)
			if nnb > 0 {
				t.q0 -= nnb
				cut(t, t, nil, false, true, "")
			}
		}
		w.previewTypeFinish(t)
		return
	}

	// 4. If range selected, cut it first (like Text.Type).
	if t.q1 > t.q0 {
		cut(t, t, nil, false, true, "")
	}

	// 5. Insert the character into the source buffer.
	t.file.InsertAt(t.q0, []rune{r})
	t.q0++
	t.q1 = t.q0

	w.previewTypeFinish(t)
}

// previewTypeFinish completes a preview-mode edit by cancelling the debounce
// timer, doing an immediate re-render, and remapping the cursor position.
func (w *Window) previewTypeFinish(t *Text) {
	// Cancel the debounce timer that the observer scheduled.
	if w.previewUpdateTimer != nil {
		w.previewUpdateTimer.Stop()
	}

	// Immediate re-render (uses incremental path via pendingEdits).
	w.UpdatePreview()

	// Remap source cursor to rendered position and update selection.
	if w.previewSourceMap != nil {
		rendStart, rendEnd := w.previewSourceMap.ToRendered(t.q0, t.q1)
		if rendStart >= 0 {
			w.richBody.SetSelection(rendStart, rendEnd)
		} else if t.q0 == t.q1 {
			// Cursor at end of content or in a gap between source map entries.
			// Fall back to the end of the rendered content.
			contentLen := w.richBody.Content().Len()
			w.richBody.SetSelection(contentLen, contentLen)
		}
	}

	w.Draw()
	if w.display != nil {
		w.display.Flush()
	}
}

// resolveImagePath resolves an image path relative to the markdown file's directory.
// If the image path is absolute (starts with /), it is returned unchanged.
// Otherwise, it is resolved relative to the directory containing basePath.
func resolveImagePath(basePath, imgPath string) string {
	// Absolute paths are returned as-is
	if filepath.IsAbs(imgPath) {
		return imgPath
	}

	// Get the directory containing the markdown file
	baseDir := filepath.Dir(basePath)

	// Join and clean the path
	resolved := filepath.Join(baseDir, imgPath)
	return filepath.Clean(resolved)
}
