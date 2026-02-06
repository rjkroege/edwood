package markdown

import (
	"strings"

	"github.com/rjkroege/edwood/rich"
)

// InlineOpts controls optional behavior of the unified inline parser.
type InlineOpts struct {
	// NoLinks disables link and image parsing. Set true when parsing
	// text inside a link label to prevent infinite recursion.
	NoLinks bool

	// SourceMap, if non-nil, receives source map entries for each
	// parsed element. Entries are appended (not replaced).
	SourceMap *[]SourceMapEntry

	// LinkMap, if non-nil, receives link entries for each parsed link.
	LinkMap *[]LinkEntry

	// SourceOffset is the byte position in the original source text
	// where `text` begins. Only used when SourceMap is non-nil.
	SourceOffset int

	// RenderedOffset is the rune position in the rendered content
	// where output spans begin. Only used when SourceMap is non-nil.
	RenderedOffset int
}

// parseInline parses inline formatting (bold, italic, code, links, images)
// within a text string and returns styled spans.
//
// Options control link/image parsing and source map generation:
//   - opts.NoLinks: set true to disable link/image recognition (used inside link text)
//   - opts.SourceMap: if non-nil, source map entries are appended
//   - opts.LinkMap: if non-nil, link entries are appended
//   - opts.SourceOffset: byte offset in source (for source map)
//   - opts.RenderedOffset: rune offset in rendered content (for source map)
func parseInline(text string, baseStyle rich.Style, opts InlineOpts) []rich.Span {
	var spans []rich.Span
	var currentText strings.Builder
	i := 0

	srcPos := opts.SourceOffset
	rendPos := opts.RenderedOffset

	flushPlain := func() {
		if currentText.Len() > 0 {
			spans = append(spans, rich.Span{
				Text:  currentText.String(),
				Style: baseStyle,
			})
			currentText.Reset()
		}
	}

	addSourceEntry := func(rendStart, rendEnd, srcStart, srcEnd int) {
		if opts.SourceMap != nil {
			*opts.SourceMap = append(*opts.SourceMap, SourceMapEntry{
				RenderedStart: rendStart,
				RenderedEnd:   rendEnd,
				SourceStart:   srcStart,
				SourceEnd:     srcEnd,
			})
		}
	}

	tracking := opts.SourceMap != nil || opts.LinkMap != nil

	for i < len(text) {
		// 1. Image: ![alt](url) — skip if NoLinks
		if !opts.NoLinks && text[i] == '!' && i+1 < len(text) && text[i+1] == '[' {
			altEnd := strings.Index(text[i+2:], "]")
			if altEnd != -1 {
				closeBracket := i + 2 + altEnd
				if closeBracket+1 < len(text) && text[closeBracket+1] == '(' {
					urlEnd := -1
					for j := closeBracket + 2; j < len(text); j++ {
						if text[j] == ')' {
							urlEnd = j
							break
						}
					}
					if urlEnd != -1 {
						flushPlain()

						altText := text[i+2 : closeBracket]
						urlPart := text[closeBracket+2 : urlEnd]
						url, title := parseURLPart(urlPart)

						placeholderText := "[Image: " + altText + "]"
						if altText == "" {
							placeholderText = "[Image]"
						}
						placeholderLen := len([]rune(placeholderText))

						imageStyle := baseStyle
						imageStyle.Fg = rich.LinkBlue
						imageStyle.Image = true
						imageStyle.ImageURL = url
						imageStyle.ImageAlt = altText
						imageStyle.ImageWidth = parseImageWidth(title)

						spans = append(spans, rich.Span{
							Text:  placeholderText,
							Style: imageStyle,
						})

						sourceLen := urlEnd - i + 1 // full ![alt](url) length
						addSourceEntry(rendPos, rendPos+placeholderLen, srcPos, srcPos+sourceLen)
						if tracking {
							rendPos += placeholderLen
							srcPos += sourceLen
						}

						i = urlEnd + 1
						continue
					}
				}
			}
			// Not a valid image, treat ! as regular text
			currentText.WriteByte(text[i])
			addSourceEntry(rendPos, rendPos+1, srcPos, srcPos+1)
			if tracking {
				rendPos++
				srcPos++
			}
			i++
			continue
		}

		// 2. Link: [text](url) — skip if NoLinks
		if !opts.NoLinks && text[i] == '[' {
			linkEnd := strings.Index(text[i+1:], "]")
			if linkEnd != -1 {
				closeBracket := i + 1 + linkEnd
				if closeBracket+1 < len(text) && text[closeBracket+1] == '(' {
					urlEnd := strings.Index(text[closeBracket+2:], ")")
					if urlEnd != -1 {
						flushPlain()

						linkText := text[i+1 : closeBracket]
						url := text[closeBracket+2 : closeBracket+2+urlEnd]
						sourceLen := 1 + linkEnd + 1 + 1 + urlEnd + 1 // [text](url)

						linkStyle := baseStyle
						linkStyle.Fg = rich.LinkBlue
						linkStyle.Link = true

						linkRenderedStart := rendPos

						if linkText == "" {
							spans = append(spans, rich.Span{
								Text:  "",
								Style: linkStyle,
							})
						} else if opts.SourceMap != nil {
							// Parse link text with source map, NoLinks to prevent recursion
							linkSpans := parseInline(linkText, linkStyle, InlineOpts{
								NoLinks:        true,
								SourceMap:      opts.SourceMap,
								SourceOffset:   srcPos + 1, // past the [
								RenderedOffset: rendPos,
							})
							spans = append(spans, linkSpans...)
							for _, span := range linkSpans {
								rendPos += len([]rune(span.Text))
							}
						} else {
							// Parse link text without source map
							linkSpans := parseInline(linkText, linkStyle, InlineOpts{
								NoLinks:        true,
								RenderedOffset: rendPos,
							})
							spans = append(spans, linkSpans...)
							if tracking {
								for _, span := range linkSpans {
									rendPos += len([]rune(span.Text))
								}
							}
						}

						linkRenderedEnd := rendPos

						// Add link entry if there's actual content
						if opts.LinkMap != nil && linkRenderedEnd > linkRenderedStart {
							*opts.LinkMap = append(*opts.LinkMap, LinkEntry{
								Start: linkRenderedStart,
								End:   linkRenderedEnd,
								URL:   url,
							})
						}

						if tracking {
							srcPos += sourceLen
						}
						i = closeBracket + 2 + urlEnd + 1
						continue
					}
				}
			}
			// Not a valid link, treat [ as regular text
			currentText.WriteByte(text[i])
			addSourceEntry(rendPos, rendPos+1, srcPos, srcPos+1)
			if tracking {
				rendPos++
				srcPos++
			}
			i++
			continue
		}

		// 3. Inline code: `text`
		if text[i] == '`' {
			end := strings.Index(text[i+1:], "`")
			if end != -1 {
				flushPlain()
				codeText := text[i+1 : i+1+end]
				codeLen := len([]rune(codeText))

				codeStyle := baseStyle
				codeStyle.Bg = rich.InlineCodeBg
				codeStyle.Code = true

				spans = append(spans, rich.Span{
					Text:  codeText,
					Style: codeStyle,
				})

				sourceLen := 1 + end + 1
				addSourceEntry(rendPos, rendPos+codeLen, srcPos, srcPos+sourceLen)
				if tracking {
					rendPos += codeLen
					srcPos += sourceLen
				}

				i = i + 1 + end + 1
				continue
			}
			// No closing ` found, treat as literal
			currentText.WriteByte(text[i])
			addSourceEntry(rendPos, rendPos+1, srcPos, srcPos+1)
			if tracking {
				rendPos++
				srcPos++
			}
			i++
			continue
		}

		// 4. Bold+italic: ***text***
		if i+2 < len(text) && text[i:i+3] == "***" {
			end := strings.Index(text[i+3:], "***")
			if end != -1 {
				flushPlain()
				innerText := text[i+3 : i+3+end]
				innerLen := len([]rune(innerText))

				s := baseStyle
				s.Bold = true
				s.Italic = true

				spans = append(spans, rich.Span{
					Text:  innerText,
					Style: s,
				})

				sourceLen := 3 + end + 3
				addSourceEntry(rendPos, rendPos+innerLen, srcPos, srcPos+sourceLen)
				if tracking {
					rendPos += innerLen
					srcPos += sourceLen
				}

				i = i + 3 + end + 3
				continue
			}
		}

		// 5. Bold: **text**
		if i+1 < len(text) && text[i:i+2] == "**" {
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				flushPlain()
				innerText := text[i+2 : i+2+end]
				innerLen := len([]rune(innerText))

				s := baseStyle
				s.Bold = true

				spans = append(spans, rich.Span{
					Text:  innerText,
					Style: s,
				})

				sourceLen := 2 + end + 2
				addSourceEntry(rendPos, rendPos+innerLen, srcPos, srcPos+sourceLen)
				if tracking {
					rendPos += innerLen
					srcPos += sourceLen
				}

				i = i + 2 + end + 2
				continue
			}
			// No closing ** found, treat as literal
			currentText.WriteString("**")
			addSourceEntry(rendPos, rendPos+2, srcPos, srcPos+2)
			if tracking {
				rendPos += 2
				srcPos += 2
			}
			i += 2
			continue
		}

		// 6. Italic: *text*
		if text[i] == '*' {
			end := strings.Index(text[i+1:], "*")
			if end != -1 {
				flushPlain()
				innerText := text[i+1 : i+1+end]
				innerLen := len([]rune(innerText))

				s := baseStyle
				s.Italic = true

				spans = append(spans, rich.Span{
					Text:  innerText,
					Style: s,
				})

				sourceLen := 1 + end + 1
				addSourceEntry(rendPos, rendPos+innerLen, srcPos, srcPos+sourceLen)
				if tracking {
					rendPos += innerLen
					srcPos += sourceLen
				}

				i = i + 1 + end + 1
				continue
			}
		}

		// 7. Regular character
		currentText.WriteByte(text[i])
		addSourceEntry(rendPos, rendPos+1, srcPos, srcPos+1)
		if tracking {
			rendPos++
			srcPos++
		}
		i++
	}

	// Flush any remaining text
	flushPlain()

	// If no spans were created, return a single span with original text
	if len(spans) == 0 {
		spans = []rich.Span{{
			Text:  text,
			Style: baseStyle,
		}}
		if opts.SourceMap != nil && text != "" {
			*opts.SourceMap = append(*opts.SourceMap, SourceMapEntry{
				RenderedStart: opts.RenderedOffset,
				RenderedEnd:   opts.RenderedOffset + len([]rune(text)),
				SourceStart:   opts.SourceOffset,
				SourceEnd:     opts.SourceOffset + len(text),
			})
		}
	}

	return spans
}
