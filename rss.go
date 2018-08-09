package main

import (
	"fmt"
	"io"

	"github.com/gorilla/feeds"
)

var feed = &feeds.Feed{
	// TODO: change this to be read from a config file
	Title:       "Jungle Coder",
	Link:        &feeds.Link{Href: "https://www.junglecoder.com/blog/"},
	Description: "The musings of a third culture coder and missionary kid",
	Author:      &feeds.Author{"Andrew Owen", "yumaikas94@gmail.com"},
}

func renderFeed(w io.Writer) error {
	ars, err := RSSArticles()
	if err != nil {
		return err
	}
	items := make([]*feeds.Item, 0)
	for _, ar := range ars {
		i := &feeds.Item{
			Title:       ar.Title,
			Link:        Article(ar).RSSLink(),
			Description: ar.HTMLContent(),
		}
		items = append(items, i)
	}
	feed.Items = items
	rss, err := feed.ToRss()
	if err != nil {
		fmt.Println(err.Error())
		return Err500
	}
	w.Write([]byte(rss))
	return nil
}

func (ar Article) RSSLink() *feeds.Link {
	// The website name is currently hardcoded. This will need to change in the future
	url := "https://www.junglecoder.com/blog/" + ar.URL
	return &feeds.Link{Href: url}
}
