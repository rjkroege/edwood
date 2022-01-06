package edwoodtest

import (
	"fmt"

	"github.com/rjkroege/edwood/draw"
)

func NiceColourName(num draw.Color) string {
	lookuptable := make(map[draw.Color]string)

	lookuptable[draw.Darkyellow] = "Darkyellow"
	lookuptable[draw.Medblue] = "Medblue"
	lookuptable[draw.Nofill] = "Nofill"
	lookuptable[draw.Notacolor] = "Notacolor"
	lookuptable[draw.Palebluegreen] = "Palebluegreen"
	lookuptable[draw.Palegreygreen] = "Palegreygreen"
	lookuptable[draw.Paleyellow] = "Paleyellow"
	lookuptable[draw.Purpleblue] = "Purpleblue"
	lookuptable[draw.Transparent] = "Transparent"
	lookuptable[draw.White] = "White"
	lookuptable[draw.Yellowgreen] = "Yellowgreen"

	if s, ok := lookuptable[num]; ok {

		return s
	}
	return fmt.Sprintf("color(%x)", num)
}
