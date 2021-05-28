package main

import "fmt"

type Observer interface {
	AddText(t *Text)
	DelText(t *Text) error
	AllText(tf func(t *Text))
	HasMultipleTexts() bool
}

type Subject struct {
	curtext *Text
	text    []*Text // [private I think]
}

func (s Subject) AddText(t *Text) {
	s.text = append(s.text, t)
	s.curtext = t
}

func (s Subject) DelText(t *Text) error {
	for i, text := range s.text {
		if text == t {
			s.text[i] = s.text[len(s.text)-1]
			s.text = s.text[:len(s.text)-1]
			if len(s.text) == 0 {
				return nil
			}
			if t == s.curtext {
				s.curtext = s.text[0]
			}
			return nil
		}
	}
	return fmt.Errorf("can't find text in File.DelText")
}

func (s Subject) AllText(tf func(t *Text)) {
	for _, t := range s.text {
		tf(t)
	}
}

func (s Subject) HasMultipleTexts() bool {
	return len(s.text) > 1
}
