package main

import (
	"fmt"
	"image"
	"time"

	"9fans.net/go/draw"
	"github.com/rjkroege/edwood/frame"
)

var scrtmp *draw.Image

func ScrSleep(dt int) {
	timer := time.NewTimer(time.Duration(dt) * time.Millisecond)
	for {
		select {
		case <-timer.C:
			return
		case <-mousectl.C:
			timer.Stop()
			return
		}
	}
}

func scrpos(r image.Rectangle, p0, p1 int, tot int) image.Rectangle {
	var (
		q image.Rectangle
		h int
	)
	q = r
	h = q.Max.Y - q.Min.Y
	if tot == 0 {
		return q
	}
	if tot > 1024*1024 {
		tot >>= 10
		p0 >>= 10
		p1 >>= 10
	}
	if p0 > 0 {
		q.Min.Y += h * p0 / tot
	}
	if p1 < tot {
		q.Max.Y -= h * (tot - p1) / tot
	}
	if q.Max.Y < q.Min.Y+2 {
		if q.Max.Y+2 <= r.Max.Y {
			q.Max.Y = q.Min.Y + 2
		} else {
			q.Min.Y = q.Max.Y - 2
		}
	}
	return q
}

func ScrlResize(display *draw.Display) {
	var err error
	scrtmp, err = display.AllocImage(image.Rect(0, 0, 32, display.ScreenImage.R.Max.Y), display.ScreenImage.Pix, false, draw.Nofill)
	if err != nil {
		panic(fmt.Sprintf("scroll alloc: %v", err))
	}
}

func (t *Text) ScrDraw(nchars int) {
	var (
		r, r1, r2 image.Rectangle
		b         *draw.Image
	)

	if t.w == nil || t != &t.w.body {
		return
	}
	if scrtmp == nil {
		ScrlResize(t.display)
	}
	r = t.scrollr
	b = scrtmp
	r1 = r
	r1.Min.X = 0
	r1.Max.X = r.Dx()
	r2 = scrpos(r1, t.org, t.org+nchars, t.file.Size())
	if !r2.Eq(t.lastsr) {
		t.lastsr = r2
		// rjk is assuming that only body Text instances have scrollers.
		b.Draw(r1, textcolors[frame.ColBord], nil, image.ZP)
		b.Draw(r2, textcolors[frame.ColBack], nil, image.ZP)
		r2.Min.X = r2.Max.X - 1
		b.Draw(r2, textcolors[frame.ColBord], nil, image.ZP)
		row.display.ScreenImage.Draw(r, b, nil, image.Pt(0, r1.Min.Y))
		// flushimage(display, 1); // BUG?
	}
}

func (t *Text) Scroll(but int) {
	var (
		p0, oldp0   int
		s           image.Rectangle
		x, y, my, h int
		first       bool
	)
	s = t.scrollr.Inset(1)
	h = s.Max.Y - s.Min.Y
	x = (s.Min.X + s.Max.X) / 2
	oldp0 = ^0
	first = true
	for {
		t.display.Flush()
		my = mouse.Point.Y
		if my < s.Min.Y {
			my = s.Min.Y
		}
		if my >= s.Max.Y {
			my = s.Max.Y
		}
		if !mouse.Point.Eq(image.Pt(x, my)) {
			t.display.MoveTo(image.Pt(x, my))
			mousectl.Read() // absorb event generated by moveto()
		}
		if but == 2 {
			y = my
			p0 = t.file.Size() * (y - s.Min.Y) / h
			if p0 >= t.q1 {
				p0 = t.Backnl(p0, 2)
			}
			if oldp0 != p0 {
				t.SetOrigin(p0, false)
			}
			oldp0 = p0
			mousectl.Read()
			if mouse.Buttons&(1<<uint(but-1)) == 0 {
				break
			}
			continue
		}
		if but == 1 {
			p0 = t.Backnl(t.org, (my-s.Min.Y)/t.fr.DefaultFontHeight())
		} else {
			p0 = t.org + t.fr.Charofpt(image.Pt(s.Max.X, my))
		}
		if oldp0 != p0 {
			t.SetOrigin(p0, true)
		}
		oldp0 = p0
		// debounce
		if first {
			t.display.Flush()
			time.Sleep(200 * time.Millisecond)
			mousectl.Mouse = <-mousectl.C
			first = false
		}
		ScrSleep(80)
		if mouse.Buttons&(1<<uint(but-1)) == 0 {
			break
		}
	}
	for mouse.Buttons != 0 {
		mousectl.Read()
	}
}
