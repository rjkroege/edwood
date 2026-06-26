package main

import (
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/theme"
)

// paletteToSpec converts a theme.Palette to the dumpfile wire format.
func paletteToSpec(p theme.Palette) *dumpfile.PaletteSpec {
	cs := func(c theme.ColorSpec) dumpfile.ColorSpec {
		return dumpfile.ColorSpec{Color: uint32(c.Color), Mix: uint32(c.Mix)}
	}
	fp := func(f theme.FramePalette) dumpfile.FramePaletteSpec {
		return dumpfile.FramePaletteSpec{
			Back:  cs(f.Back),
			High:  cs(f.High),
			Bord:  cs(f.Bord),
			Text:  cs(f.Text),
			HText: cs(f.HText),
			Tick:  cs(f.Tick),
		}
	}
	return &dumpfile.PaletteSpec{
		Tag:  fp(p.Tag),
		Text: fp(p.Text),
		Ui: dumpfile.UiPaletteSpec{
			ModButton: cs(p.Ui.ModButton),
			ColButton: cs(p.Ui.ColButton),
			But2:      cs(p.Ui.But2),
			But3:      cs(p.Ui.But3),
		},
	}
}

// paletteFromSpec converts a dumpfile PaletteSpec to a theme.Palette.
func paletteFromSpec(spec *dumpfile.PaletteSpec) theme.Palette {
	cs := func(s dumpfile.ColorSpec) theme.ColorSpec {
		return theme.ColorSpec{Color: draw.Color(s.Color), Mix: draw.Color(s.Mix)}
	}
	fp := func(s dumpfile.FramePaletteSpec) theme.FramePalette {
		return theme.FramePalette{
			Back:  cs(s.Back),
			High:  cs(s.High),
			Bord:  cs(s.Bord),
			Text:  cs(s.Text),
			HText: cs(s.HText),
			Tick:  cs(s.Tick),
		}
	}
	return theme.Palette{
		Tag:  fp(spec.Tag),
		Text: fp(spec.Text),
		Ui: theme.UiPalette{
			ModButton: cs(spec.Ui.ModButton),
			ColButton: cs(spec.Ui.ColButton),
			But2:      cs(spec.Ui.But2),
			But3:      cs(spec.Ui.But3),
		},
	}
}
