package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"

	md "github.com/russross/blackfriday"
	arts "github.com/yumaikas/blogserv/blogArticles"
	"github.com/yumaikas/blogserv/config"
	die "github.com/yumaikas/golang-die"
)

var (
	blogTemps template.Template
)

//Take the first two lines and use it for an article preview
func Preview(s string) template.HTML {
	var preview string
	for idx, tx := range strings.Split(s, "\n") {
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
	rss := template.HTMLEscapeString(ar.Content)
	return template.HTML(rss)
}

//This content needs to be trusted
func (ar *Article) HTMLContent() template.HTML {
	ar.Content = string(md.MarkdownCommon([]byte(ar.Content)))
	ar.Content = strings.Replace(ar.Content, `<code class="`, `<code class="prettyprint lang-`, -1)
	ar.Content = strings.Replace(ar.Content, `<code>`, `<code class="prettyprint">`, -1)
	return template.HTML(ar.Content)
}

type articleList []arts.Article
type rssfeed []arts.Article

var (
	goarticle  chan int
	goblogroll chan int
	gorssFeed  chan int
)

func (ar *Article) render(out io.Writer) (err error) {
	err = blogTemps.ExecuteTemplate(out, "blogPost", ar)
	return
}

func (ars articleList) render(out io.Writer) (err error) {
	err = blogTemps.ExecuteTemplate(out, "blogRoll", ars)
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

var reset chan int

func listenForAdmin() {
	reset = make(chan int)
	l, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		panic(err.Error())
	}
	for {
		c, err := l.Accept()
		if err != nil {
			c.Close()
			continue
		}
		scn := bufio.NewScanner(c)
		if scn.Scan() {
			t := scn.Text()
			switch t {
			case "reset":
				reset <- 0
				fmt.Fprintln(c, "Reset sent")
			default:
			}
		}
		c.Close()
	}
}

func template_init() {
	blogTemps = template_load()
	timeout := time.Tick(5 * time.Minute)
	for {
		select {
		case <-reset:
			fmt.Println("resetting config and templates")
			config.ReloadConfig()
			blogTemps = template_load()
		case <-timeout:
			fmt.Println("refreshing templates")
			blogTemps = template_load()
		}
	}
}

//Prepare the templates for the server, then test
//TODO: generalize this so that it uses a config file to get the list of templates used, or just walks a directory.
func template_load() template.Template {

	funcs := template.FuncMap{
		"splitUpper": spaceTitleCase,
		"preview":    Preview,
		"isDraft":    arts.IsDraft,
	}
	temps, err := template.New("sidebar").Funcs(funcs).Parse("")
	die.OnErr(err)
	// This function will make a fatal log if it fails, exiting the process
	loadTemplate := func(file string) {
		//template path has a trailing slash so that file name
		//doesn't need to have leading one
		temp, err := ioutil.ReadFile(config.TemplatePath() + file)
		templ := string(temp)
		die.OnErr(err)
		//No name is needed here as the templates are expected to supply their own names.
		_, err = temps.Parse(templ)
		die.OnErr(err)
	}

	loadTemplate("sidebar.gohtml")
	loadTemplate("blogRoll.gohtml")
	loadTemplate("blogPost.gohtml")
	loadTemplate("Login.gohtml")
	loadTemplate("editor.gohtml")
	loadTemplate("creator.gohtml")
	loadTemplate("serverError.gohtml")

	return *temps
}
