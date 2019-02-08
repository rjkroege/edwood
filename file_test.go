package main

import "testing"

func TestDelText(t *testing.T) {
	f := &File{
		text: []*Text{{}, {}, {}, {}, {}},
	}
	t.Run("Nonexistent", func(t *testing.T) {
		err := f.DelText(&Text{})
		if err == nil {
			t.Errorf("expected panic when deleting nonexistent text")
		}
	})
	for i := len(f.text) - 1; i >= 0; i-- {
		text := f.text[i]
		err := f.DelText(text)
		if err != nil {
			t.Errorf("DelText of text at index %d failed: %v", i, err)
			continue
		}
		if got, want := len(f.text), i; got != want {
			t.Fatalf("DelText resulted in text of length %v; expected %v", got, want)
		}
		for i, t1 := range f.text {
			if t1 == text {
				t.Fatalf("DelText did not delete correctly at index %v", i)
			}
		}
	}
}
