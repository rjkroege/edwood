package main

// Helpful utility deried from https://github.com/riesinger/golorem to generate test text.

import (
	"fmt"
	"math/rand"
	"strings"
)

var wordsList = []string{
	"ipsum", "semper", "habeo", "duo", "ut", "vis", "aliquyam", "eu", "splendide", "Ut", "mei", "eteu", "nec", "antiopam", "corpora", "kasd", "pretium", "cetero", "qui", "arcu", "assentior", "ei", "his", "usu", "invidunt", "kasd", "justo", "ne", "eleifend", "per", "ut", "eam", "graeci", "tincidunt", "impedit", "temporibus", "duo", "et", "facilisis", "insolens", "consequat", "cursus", "partiendo", "ullamcorper", "Vulputate", "facilisi", "donec", "aliquam", "labore", "inimicus", "voluptua", "penatibus", "sea", "vel", "amet", "his", "ius", "audire", "in", "mea", "repudiandae", "nullam", "sed", "assentior", "takimata", "eos", "at", "odio", "consequat", "iusto", "imperdiet", "dicunt", "abhorreant", "adipisci", "officiis", "rhoncus", "leo", "dicta", "vitae", "clita", "elementum", "mauris", "definiebas", "uonsetetur", "te", "inimicus", "nec", "mus", "usu", "duo", "aenean", "corrumpit", "aliquyam", "est", "eum",
}

func getRandomWord() string {
	return wordsList[rand.Intn(len(wordsList))]
}

func generateWords(length int) string {
	var b strings.Builder
	b.WriteString("Lorem ")
	for i := 0; i < length-1; i++ {
		b.WriteString(getRandomWord())
		b.WriteString(" ")
	}
	return b.String()
}

func generateParagraphs(count, length int, separator string) string {
	result := ""
	if length == 0 {
		for i := 0; i < count; i++ {
			result += fmt.Sprintf("%s%s", generateWords(10), separator)
		}
		return result
	} else {
		for i := 0; i < count; i++ {
			result += fmt.Sprintf("%s%s", generateWords(length), separator)
		}
		return result
	}
}
