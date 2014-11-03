package main

import (
	"fmt"
	"io/ioutil"

	"github.com/russross/blackfriday"
	secret "github.com/yumaikas/blogserv/config"
	//"os"
	"bytes"
	"flag"
	"strings"
)

var root = secret.PostPath()

func main() {
	flag.Parse()
	dumpDb()
	readArticles()
}
func dumpDb() {
	ars, err := getArticles()
	if err != nil {
		panic(fmt.Sprintf("Error: %s", err.Error()))
	}
	NL := "\n"
	for _, a := range ars {

		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, `Title:"%s"`+NL, a.Title)
		fmt.Fprintf(buf, `URL:"%s"`+NL, a.URL)
		fmt.Fprintf(buf, `PublishStage:"Publish"`+NL)
		fmt.Fprintf(buf, `Text:{`+NL)
		fmt.Fprintf(buf, `%s`, a.Content)
		fmt.Fprintf(buf, NL+`}:Text`)
		filePath := root + strings.Replace(a.URL, ":", "", -1) + ".post"
		err = ioutil.WriteFile(filePath, buf.Bytes(), 0644)
		if err != nil {
			panic(fmt.Sprintf("Couldn't write post %s because: %s", filePath, err.Error()))
		}
	}
}

func readArticles() {

	// file := os.Open(root + `\dump.html`)
	files, err := ioutil.ReadDir(root)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	articles := make([]*article, 0)
	errs := make([]error, 0)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "mdown") {
			contents, err := ioutil.ReadFile(fmt.Sprintf(`%s/%s`, root, file.Name()))
			if err != nil {
				fmt.Print(err.Error())
				return
			}
			if ar, err := parseFile(string(contents)); err == nil {
				articles = append(articles, ar)
			} else {
				errs = append(errs, fmt.Errorf("File:%s Errors:\n%s", file.Name(), err.Error()))
			}
		}
	}
	if errs != nil && len(errs) > 0 {
		fmt.Println(errs)
		return
	}
	dbArticles, err := getArticles()
	for _, ar := range articles {
		htmlout := blackfriday.MarkdownCommon([]byte(ar.Content))
		ar.Content = string(htmlout)
		fmt.Printf("%s, Publish Stage: |%s|\n", ar.Title, ar.PublishStage)
		if _, ok := dbArticles[ar.URL]; ok && ar.PublishStage == "Publish" {
			update(*ar)
		} else if ar.PublishStage == "Publish" {
			fmt.Println(ar)
			insert(*ar)
		}
		// Here we deliberatly do nothing
	}
}

type article struct {
	Title, URL, Content, PublishStage string
}
