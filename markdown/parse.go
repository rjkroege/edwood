package markdown

import "github.com/rjkroege/edwood/rich"

// Parse converts markdown text to styled rich.Content.
// This is a minimal implementation that currently only handles plain text.
func Parse(text string) rich.Content {
	if text == "" {
		return rich.Content{}
	}
	return rich.Plain(text)
}
