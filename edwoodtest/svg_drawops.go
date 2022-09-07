package edwoodtest

// TODO(rjk): write up some entry points

import (
	"html/template"
	"image"
	"io"
	"log"
	"strings"
)

var tmpl *template.Template

// init creates a single template with multiple sub-templates.
func init() {
	tmpl = template.New("svgout")
	template.Must(tmpl.New("Bytes").Parse(bytetemplate))
	template.Must(tmpl.New("Fill").Parse(filltemplate))
	template.Must(tmpl.New("Boundingbox").Parse(boundingboxtemplate))
	template.Must(tmpl.New("Blit").Parse(blittemplate))
	template.Must(tmpl.New("Final").Parse(finalfiletemplate))
}

const bytetemplate = `<g id="draw{{.Id}}">
	<use href="#draw{{.SrcId}}" />
	{{range .Glyphs}}<text x="{{.X}}" y="{{.Y}}" fill="black" class="small">{{.R}}</text>
	{{end -}}
	</g>`

type Glyphy struct {
	X int
	Y int
	R string
}

type Bytesargs struct {
	Id     int
	SrcId  int
	Glyphs []Glyphy
}

// TODO(rjk): plumb the colour
// TODO(rjk): structure this more nicely
// TODO(rjk): generalize font metrics
// TODO(rjk): use the font metrics in the generated SVG
func bytessvg(id int, sp image.Point, b []byte) string {
	s := string(b)

	glyphs := make([]Glyphy, 0, len(s))
	x := sp.X
	for _, r := range s {
		glyphs = append(glyphs, Glyphy{
			R: string(r),
			X: x + 2,
			// 9p uses the top corner, not the baseline.
			Y: sp.Y + fheight - 2,
		})

		// TODO(rjk): Configure this. Need metrics
		x += fwidth
	}

	byteargs := Bytesargs{
		Id:     id,
		SrcId:  id - 1,
		Glyphs: glyphs,
	}

	swr := new(strings.Builder)
	if err := tmpl.ExecuteTemplate(swr, "Bytes", byteargs); err != nil {
		log.Printf("can't run the template on %v because %v\n", byteargs, err)
	}

	return swr.String()
}

// TODO(rjk): refactor this together with the other code.
type Fillargs struct {
	Id    int
	SrcId int
	Box   image.Rectangle
	Rect  image.Rectangle
}

const filltemplate = `<g id="draw{{.Id}}">
	<use href="#draw{{.SrcId}}" />
	<rect x="{{.Rect.Min.X}}" y="{{.Rect.Min.Y}}" width="{{.Rect.Dx}}" height="{{.Rect.Dy}}" fill="#ffffdd"/>
	</g>`

func fillsvg(id int, rect, box image.Rectangle) string {
	fillargs := Fillargs{
		Id:    id,
		SrcId: id - 1,
		Rect:  rect,
		Box:   box,
	}

	swr := new(strings.Builder)
	if err := tmpl.ExecuteTemplate(swr, "Fill", fillargs); err != nil {
		log.Printf("can't run the template on fill because %v\n", err)
	}

	return swr.String()
}

type Blitargs struct {
	Id         int
	SrcId      int
	BlitOffset int
	Src        image.Rectangle
	Dest       image.Point
}

const blittemplate = `<use href="#draw{{.SrcId}}" />
<rect x="{{.Src.Min.X}}" y="{{.Src.Min.Y}}" width="{{.Src.Dx}}" height="{{.Src.Dy}}" fill="none" stroke="red"/>
<g transform="translate({{.BlitOffset}}, 0)">
	<g id="draw{{.Id}}">
		<use href="#draw{{.SrcId}}" />
		<clipPath id="draw{{.Id}}_blitsource">
			<!-- This is the rectangle corresponding to the blit source -->
			<rect x="{{.Src.Min.X}}" y="{{.Src.Min.Y}}" width="{{.Src.Dx}}" height="{{.Src.Dy}}" />
		</clipPath>
		<use href="#draw{{.SrcId}}" clip-path="url(#draw{{.Id}}_blitsource)" x="{{.Dest.X}}" y="{{.Dest.Y}}" />
	</g>
</g>
`

// blitsvg returns an SVG fragment visulizing a blit operation.
func blitsvg(id int, src image.Rectangle, pt image.Point, offset int) string {
	blitargs := Blitargs{
		Id:         id,
		SrcId:      id - 1,
		Src:        src,
		Dest:       pt.Sub(src.Min),
		BlitOffset: offset,
	}

	swr := new(strings.Builder)
	if err := tmpl.ExecuteTemplate(swr, "Blit", blitargs); err != nil {
		log.Printf("can't run the template on Blit because %v\n", err)
	}

	return swr.String()
}

const boundingboxtemplate = `<g id="draw{{.Id}}">
	<rect x="{{.Box.Min.X}}" y="{{.Box.Min.Y}}" width="{{.Box.Dx}}" height="{{.Box.Dy}}" fill="none" stroke="black"/>
	</g>`

// TODO(rjk): There is opportunity to refactor this quite extensively.
type Boxargs struct {
	Id  int
	Box image.Rectangle
}

// boundingboxsvg generates a starting point visualization: the region of
// interest.
func boundingboxsvg(id int, box image.Rectangle) string {
	boxargs := Boxargs{
		Id:  id,
		Box: box,
	}

	swr := new(strings.Builder)
	if err := tmpl.ExecuteTemplate(swr, "Boundingbox", boxargs); err != nil {
		log.Printf("can't run the template on Boundingbox because %v\n", err)
	}

	return swr.String()
}

// TODO(rjk): Note that I could configure the font here from fwidth, fheight.
// Note how I can introduce variables. Note also how I can invoke a
// function. The function is just an argument to the template execution.
// The use of call is how I can make arbitrary functions for content
// generation.
const finalfiletemplate = `<html lang="en-US">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width">
</head>
<body>
<svg viewBox="{{.ViewBox.Min.X}} {{.ViewBox.Min.Y}} {{.ViewBox.Max.X}} {{.ViewBox.Max.Y}}" xmlns="http://www.w3.org/2000/svg">

<style>
	.small { font: 8px sans-serif; }
</style>

{{- $boxsize := .ScreenBox.Dy -}}
{{- $vertfunc := .VertOffset }}
{{range $index, $element := .Fragments}}
<g transform="translate(0, {{call $vertfunc $index $boxsize}})">
	<g transform="translate(0, -2)">
		<text x="0" y="0" fill="black" class="small">{{$element.Annotation}}</text>
	</g>
	{{$element.SVG}}
</g>
{{end}}
</svg>
</body>
</html>
`

// verticaloffset is a helper function used inside the template
// execution. Note above how it's passed into the template as VertOffset
// and then used via a call template statement.
func verticaloffset(i, boxheight int) int {
	return i * (boxheight + padding)
}

// TODO(rjk): There is opportunity to refactor this.
type Finalfileargs struct {
	// Becomes the viewBox property of the generated SVG.
	ViewBox image.Rectangle

	// The previous draw ops as separate strings.
	Fragments []AnnotatedFragments

	// The rectangle of interest. For example, in a test of the frame code,
	// this might be the rectangle corresponding to the text area.
	ScreenBox image.Rectangle

	// Helper function used to move down in the visualization between
	// successive draw operations.
	VertOffset func(int, int) int
}

type AnnotatedFragments struct {
	// A descriptive annotation for this fragment.
	Annotation string

	// The chunk of SVG corresponding to this fragment.
	SVG template.HTML
}

const (
	padding   = 40 // edge padding
	blitspace = 30 // distance between the blit src and blit dest
)

// singlesvgfile writes a single HTML file to w containing a scrollable
// sequence of subops. rectofi is the rectangle of interest to consider.
func singlesvgfile(w io.Writer, subops, annotations []string, rectofi image.Rectangle) error {
	annotatedfrags := make([]AnnotatedFragments, 0, len(subops))

	for i, s := range subops {
		annotatedfrags = append(annotatedfrags, AnnotatedFragments{
			Annotation: annotations[i],
			SVG:        template.HTML(s),
		})
	}

	finalargs := Finalfileargs{
		ViewBox: image.Rect(
			rectofi.Min.X-padding,
			rectofi.Min.Y-padding,
			2*rectofi.Dx()+blitspace+2*padding,
			(rectofi.Dy()+padding)*len(subops),
		),
		Fragments:  annotatedfrags,
		ScreenBox:  rectofi,
		VertOffset: verticaloffset,
	}

	return tmpl.ExecuteTemplate(w, "Final", finalargs)
}
