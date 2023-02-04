package main

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"strings"
)

// parse single file for links
func parse(sourceFile, contentRoot string) []Link {
	// read file
	source, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		panic(err)
	}

	// parse md
	var links []Link
	fmt.Printf("[Parsing note] %s => \n", trim(sourceFile, contentRoot, ".md"))

	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		panic(err)
	}

	doc, err := goquery.NewDocumentFromReader(&buf)
	var n int
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		target, ok := s.Attr("href")
		if !ok {
			target = "#"
		}

		target = processTarget(sourceFile, target, contentRoot)
		source := processSource(trim(sourceFile, contentRoot, ".md"))

		fmt.Printf("find target: %s, source: %s, text: %s\n", target, source, text)

		// fmt.Printf("  '%s' => %s\n", source, target)
		links = append(links, Link{
			Source: source,
			Target: target,
			Text:   text,
		})
		n++
	})
	fmt.Printf("[Parsing note] %s => find %d links \n", trim(sourceFile, contentRoot, ".md"), n)

	return links
}
