package main

//This is a simple http webserver.
import (
	secret "blogserv_secret"
	"bytes"
	"fmt"
	"net/http"
	"runtime"
)

var (
	webRoot  string       = secret.WebRoot
	fileRoot http.Handler = http.FileServer(http.Dir(webRoot))
)

type Page struct {
	Title string
	Body  []byte
}

func init() {
	//If any of these functions fail, we aren't ready for the server to run.
	//They all call log.Fatal(), so the user will get a warning when they run.
	akismet_init()
	go template_init()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//Force a defualt main page
	http.HandleFunc("/index.html", home)
	http.HandleFunc("/blog", home)
	http.HandleFunc("/blog/", getArticle)
	http.HandleFunc("/blog/feed.xml", getFeed)
	http.HandleFunc("/submitComment/", postComment)
	http.HandleFunc("/", root)

	//For production use port 80
	err := http.ListenAndServe(":80", logMux)
	//For testing use port 8080
	//err := http.ListenAndServe(":8080", logMux)
	if err != nil {
		fmt.Println(err.Error())
	}
}
func root(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) > 1 {
		fileRoot.ServeHTTP(w, r)
		return
	}
	home(w, r)
}
func home(w http.ResponseWriter, r *http.Request) {
	b := new(bytes.Buffer)
	ars, err := HTMLArticles()
	if err != nil {
		w.WriteHeader(500)
		ars.render(w)
		fmt.Fprintf(w, "%s", err.Error())
		return
	}
	err = ars.render(b)
	if err != nil {
		w.WriteHeader(500)
		b.WriteTo(w)
		fmt.Fprintf(w, "%s", Err500.Error())
		return
	}

	b.WriteTo(w)
}

func getArticle(w http.ResponseWriter, r *http.Request) {
	articleTitle := r.URL.Path[len("/blog/"):]
	if len(articleTitle) == 0 {
		home(w, r)
		return
	}
	ar, err := fillArticle(articleTitle)
	if err != nil {
		fmt.Fprintln(w, "An error occurred while attempting to fetch the article")
		return
	}
	err = ar.render(w)
	if err != nil {
		fmt.Fprintf(w, "An error occurred while attempting to parse the article %e", err)
		return
	}
}

func getFeed(w http.ResponseWriter, r *http.Request) {
	b := new(bytes.Buffer)
	if err := renderFeed(b); err != nil {
		w.WriteHeader(500)
		w.Write([]byte(Err500.Error()))
		return
	}
	b.WriteTo(w)
}
