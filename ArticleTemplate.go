package main

import (
	secret "blogserv_secret"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"strings"
	"time"
)

//The Article Type holds
type Article struct {
	Title, URL, content string
	//This conent doesn't come from my typing, it shouldn't be trusted.
	Comments []Comment
}

//Take the first two lines and use it for an article preview
func (ar *Article) Preview() template.HTML {
	var preview string
	for idx, tx := range strings.Split(ar.content, "\n") {
		//Only take 2 lines
		if idx > 2 {
			break
		}
		preview += tx + "\n"
	}
	preview += " "
	return template.HTML(preview)
}

//Html escape the content of the article so that RSS readers can parse it.
func (ar *Article) RssHTML() template.HTML {
	rss := template.HTMLEscapeString(ar.content)
	return template.HTML(rss)
}

func (ar *Article) Content() template.HTML {
	return template.HTML(ar.content)
}

type articleList []Article
type rssfeed []Article

var (
	article  template.Template
	blogRoll template.Template
	rssFeed  template.Template
)

func (f rssfeed) render(out io.Writer) (err error) {
	err = rssFeed.Execute(out, f)
	return
}

func (ar *Article) render(out io.Writer) (err error) {
	err = article.Execute(out, ar)
	return
}

func (ars articleList) render(out io.Writer) (err error) {
	err = blogRoll.Execute(out, ars)
	return
}

//This is so that URL's can be TitleCased but page titles are spaced
func spaceTitleCase(str string) string {
	var newTitle []rune
	for _, c := range str {
		if c >= 'A' && c <= 'Z' {
			newTitle = append(newTitle, ' ')
		}
		newTitle = append(newTitle, c)
	}
	return string(newTitle)
}
func template_init() {
	article, blogRoll = template_load()
	timeout := time.Tick(5 * time.Minute)
	for {
		select {
		case <-timeout:
			fmt.Println("refreshing templates")
			article, blogRoll = template_load()
		}
	}
}

//Prepare the templates for the server, then test
func template_load() (template.Template, template.Template) {
	//This function will make a fatal log if it fails, exiting the process
	loadTemplate := func(file string) string {
		//template path has a trailing slash so that file name
		//doesn't need to have leading one
		temp, err := ioutil.ReadFile(secret.TemplatePath + file)
		templ := string(temp)
		if err != nil {
			//Without templates, the server can't run
			panic(fmt.Errorf("The template load failed: %s", err.Error()))
		}
		return templ
	}

	funcs := template.FuncMap{
		"splitUpper": spaceTitleCase,
	}
	bp_temp := loadTemplate("blogPost.html")
	br_temp := loadTemplate("blogRoll.html")
	parseTemplate := func(title, text string) template.Template {
		//Again, this is a fatal log if we have a failure.
		templ, err := template.New(title).Funcs(funcs).Parse(text)
		if err != nil {
			panic(fmt.Sprintf("The template load faild %s", err.Error()))
		}
		return *templ
	}
	p_article := parseTemplate("ArticleTemplate", bp_temp)
	p_blogRoll := parseTemplate("BlogRollTemplate", br_temp)
	return p_article, p_blogRoll
}
