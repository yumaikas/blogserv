package main

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func NL() string {
	return "\n"
}

var (
	// Individual tags
	tg = regexp.MustCompile(`(?m)^(\w+):"([^"]([^"]|\")*)"$`)
	// Match the article text
	tx_re = regexp.MustCompile(`(?s)Text:\{(.+)\}:Text`)
	// Match the tags before the article text (needs to be split by newlines for tg to match)
	tags = regexp.MustCompile(`(?s)^(.+)Text:{`)
)

// c is the content of the article from the file.
func parseFile(c string) (*article, error) {

	if !tags.MatchString(c) {
		panic("No tags for article and URL!")
	}
	a := new(article)
	lines := tags.FindStringSubmatch(c)
	for _, l := range strings.Split(lines[1], NL()) {
		l = strings.TrimSpace(l)
		if tg.MatchString(l) {
			m := tg.FindStringSubmatch(l)
			// assign based on tag
			switch m[1] {
			case "Title":
				fmt.Printf("Title: %v\n", m[2])
				a.Title = m[2]
			case "URL":
				a.URL = m[2]
			case "PublishStage":
				fmt.Printf("PublishStage: %v\n", m[2])
				a.PublishStage = m[2]
			}
			// For debugging
			// fmt.Printf("tag: %q, value: %q\n", m[1], m[2])
		}
	}
	// Grab all the text.
	txt := tx_re.FindStringSubmatch(c)
	a.Content = txt[1]
	// Validate the article
	errBuf := new(bytes.Buffer)
	if a.URL == "" {
		errBuf.WriteString("Article URL is empty.\n")
	}
	if a.Title == "" {
		errBuf.WriteString("No title for the article.\n")
	}
	if a.Content == "" {
		errBuf.WriteString("No content for article.\n")
	}
	if errBuf.Len() > 0 {
		return nil, errors.New(errBuf.String())
	}
	// For debugging
	// fmt.Println(txt[1])
	return a, nil
}
