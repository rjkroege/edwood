package main

// All the sub-functions needed to implement typing are in this file.

// tagdown expands the tag to show all of the text.
func (t *Text) tagdown() {
	if t.what != Tag {
		return
	}

	if !t.w.tagexpand {
		t.w.tagexpand = true
		t.w.Resize(t.w.r, false, true)
	}

}
