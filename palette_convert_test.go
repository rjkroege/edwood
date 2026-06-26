package main

import (
	"testing"

	"github.com/rjkroege/edwood/theme"
)

// TestPaletteConvertRoundtrip checks that every built-in palette survives a
// paletteToSpec → paletteFromSpec round-trip with no loss of colour data.
func TestPaletteConvertRoundtrip(t *testing.T) {
	palettes := []struct {
		name string
		p    theme.Palette
	}{
		{"acme (Light)", theme.Light},
		{"vampira (Dark)", theme.Dark},
		{"solarizedlight", theme.SolarizedLight},
		{"solarizeddark", theme.SolarizedDark},
	}

	for _, tc := range palettes {
		t.Run(tc.name, func(t *testing.T) {
			spec := paletteToSpec(tc.p)
			got := paletteFromSpec(spec)

			checkFrame := func(role string, want, have theme.FramePalette) {
				t.Helper()
				if want.Back != have.Back {
					t.Errorf("%s Back: want %v got %v", role, want.Back, have.Back)
				}
				if want.High != have.High {
					t.Errorf("%s High: want %v got %v", role, want.High, have.High)
				}
				if want.Bord != have.Bord {
					t.Errorf("%s Bord: want %v got %v", role, want.Bord, have.Bord)
				}
				if want.Text != have.Text {
					t.Errorf("%s Text: want %v got %v", role, want.Text, have.Text)
				}
				if want.HText != have.HText {
					t.Errorf("%s HText: want %v got %v", role, want.HText, have.HText)
				}
				if want.Tick != have.Tick {
					t.Errorf("%s Tick: want %v got %v", role, want.Tick, have.Tick)
				}
			}

			checkFrame("Tag", tc.p.Tag, got.Tag)
			checkFrame("Text", tc.p.Text, got.Text)

			if tc.p.Ui.ModButton != got.Ui.ModButton {
				t.Errorf("Ui.ModButton: want %v got %v", tc.p.Ui.ModButton, got.Ui.ModButton)
			}
			if tc.p.Ui.ColButton != got.Ui.ColButton {
				t.Errorf("Ui.ColButton: want %v got %v", tc.p.Ui.ColButton, got.Ui.ColButton)
			}
			if tc.p.Ui.But2 != got.Ui.But2 {
				t.Errorf("Ui.But2: want %v got %v", tc.p.Ui.But2, got.Ui.But2)
			}
			if tc.p.Ui.But3 != got.Ui.But3 {
				t.Errorf("Ui.But3: want %v got %v", tc.p.Ui.But3, got.Ui.But3)
			}
		})
	}
}

// TestPaletteConvertMixed verifies that mixed() ColorSpecs (non-zero Mix field)
// survive the round-trip without the Mix component being silently zeroed.
func TestPaletteConvertMixed(t *testing.T) {
	// theme.Light uses mixed() for Tag.Back and Text.Back.
	spec := paletteToSpec(theme.Light)
	if spec.Tag.Back.Mix == 0 {
		t.Errorf("Tag.Back.Mix should be non-zero after paletteToSpec (Light uses mixed())")
	}
	if spec.Text.Back.Mix == 0 {
		t.Errorf("Text.Back.Mix should be non-zero after paletteToSpec (Light uses mixed())")
	}

	got := paletteFromSpec(spec)
	if got.Tag.Back != theme.Light.Tag.Back {
		t.Errorf("Tag.Back round-trip: want %v got %v", theme.Light.Tag.Back, got.Tag.Back)
	}
	if got.Text.Back != theme.Light.Text.Back {
		t.Errorf("Text.Back round-trip: want %v got %v", theme.Light.Text.Back, got.Text.Back)
	}
}
