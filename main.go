package main

//This is a simple http webserver.
import (
	"bytes"
	"fmt"
	"mime"
	"net/http"

	"github.com/yumaikas/blogserv/WebAdmin"
	arts "github.com/yumaikas/blogserv/blogArticles"
	"github.com/yumaikas/blogserv/config"
)

var (
	webRoot  string       = config.WebRoot()
	fileRoot http.Handler = http.FileServer(http.Dir(webRoot))
)

type Page struct {
	Title string
	Body  []byte
}

func init() {
	//pIf any of these functions fail, we aren't ready for the server to run.
	//They all call log.Fatal(), so the user will get a warning when they run.
	//akismet_init()

	go template_init()
	go listenForAdmin()
}

func addClickOnceMimeTypes() {
	//These extensions are to make ClickOnce deployments work properly, so that
	//Chrome and Firefox will work corretly with the.
	mime.AddExtensionType(".application", "application/x-ms-application")
	mime.AddExtensionType(".manifest", "application/x-ms-manifest")
	mime.AddExtensionType(".deploy", "application/octet-stream")
	mime.AddExtensionType(".msp", "application/octet-stream")
	mime.AddExtensionType(".msu", "application/octet-stream")
	mime.AddExtensionType(".vsto", "application/x-ms-vsto")
	mime.AddExtensionType(".xaml", "application/xaml+xml")
	mime.AddExtensionType(".xbap", "application/x-ms-xbap")
}
func main() {
	//Needs to be called inside main() for some reason. I suspect that it has
	//something to do with initization order, but am not sure.
	addClickOnceMimeTypes()
	//Force a defualt main page
	http.HandleFunc("/", root)
	http.HandleFunc("/index.html", home)
	http.HandleFunc("/blog", home)
	http.HandleFunc("/blog/", getArticle)
	http.HandleFunc("/blog/feed.xml", getFeed)
	http.HandleFunc("/submitComment/", postComment)
	http.HandleFunc("/api/", api)
	http.HandleFunc("/blog/login", loginRoute)
	http.HandleFunc("/admin/login", performLogin)
	http.HandleFunc("/admin/logout", performLogout)
	http.HandleFunc("/admin/edit/", edit)

	//For production use port 80
	err := http.ListenAndServe(":8080", logMux)
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
	isAdmin := WebAdmin.AttemptAuth(w, r)

	fmt.Println(r.Cookies())
	fmt.Println("Admin:", isAdmin)
	for idx, _ := range ars {
		ars[idx].IsAdmin = isAdmin
	}
	for _, ar := range ars {
		fmt.Println(ar.IsAdmin)
	}
	if err != nil {
		w.WriteHeader(500)
		ars.render(w)
		fmt.Fprintf(w, "%s", err.Error())
		return
	}
	err = ars.render(b)
	if err != nil {
		fmt.Printf("An error occurred while rendering templates %s", err.Error())
		w.WriteHeader(500)
		b.WriteTo(w)
		fmt.Fprintf(w, "%s", Err500.Error())
		return
	}

	b.WriteTo(w)
}

func getArticle(w http.ResponseWriter, r *http.Request) {
	isAdmin := WebAdmin.AttemptAuth(w, r)
	articleTitle := r.URL.Path[len("/blog/"):]
	if len(articleTitle) == 0 {
		home(w, r)
		return
	}
	ar, err := fillArticle(articleTitle)
	ar.IsAdmin = isAdmin
	fmt.Println(isAdmin)
	if err != nil {
		fmt.Fprintln(w, "An error occurred while attempting to fetch the article")
		return
	}

	if ar.PublishStage != "Published" && isAdmin {
		http.Redirect(w, r, "/admin/edit/"+articleTitle, 303)
		return
	}
	err = ar.render(w)
	if err != nil {
		fmt.Fprintf(w, "An error occurred while attempting to parse the article %e", err)
		return
	}
}

func performLogin(w http.ResponseWriter, r *http.Request) {
	if WebAdmin.AttemptAuth(w, r) {
		WebAdmin.AddNameCookie(w, r)
		http.Redirect(w, r, "/blog/", 303)
	} else {
		fmt.Println("failed attemped auth")
		w.WriteHeader(http.StatusForbidden)
	}
}

func performLogout(w http.ResponseWriter, r *http.Request) {
	//FIXME: get rid of hard coded string here, by reading user ID from cookie.
	WebAdmin.ClearToken("yumaikas")

	c, err := r.Cookie("authToken")
	if err == nil {
		c.MaxAge = -1
		c.Value = ""
	}
	http.SetCookie(w, c)
	http.Redirect(w, r, "/blog/", 303)
}

func edit(w http.ResponseWriter, r *http.Request) {
	//Fail fast
	if !WebAdmin.AttemptAuth(w, r) {
		w.WriteHeader(404)
		w.Write([]byte(Err404.Error()))
		return
	}
	if r.Method == "GET" {
		editSetup(w, r)
		return
	}
	if r.Method == "POST" {
		editSubmit(w, r)
		return
	}
	fmt.Println("Invalid method for edit URL.")
	w.WriteHeader(404)
	fmt.Fprint(w, Err404.Error())
}

func editSubmit(w http.ResponseWriter, r *http.Request) {
	articleTitle := r.URL.Path[len("/admin/edit/"):]
	ar, err := fillArticle(articleTitle)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, Err500.Error())
		return
	}
	r.ParseForm()

	title := r.PostFormValue("Title")
	article := r.PostFormValue("article")
	art := arts.Article(ar)
	art.Content = article
	art.Title = title
	art.PublishStage = "Draft"
}

//This function needs to be auth verified before calling.
func editSetup(w http.ResponseWriter, r *http.Request) {

	articleTitle := r.URL.Path[len("/admin/edit/"):]
	ar, err := fillArticle(articleTitle)

	if err == arts.ErrArticleNotFound {
		w.WriteHeader(404)
		w.Write([]byte("Article not found!"))
		return
	} else if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(Err500.Error()))
		fmt.Println(err)
		return
	}
	data := struct {
		Content, Title, URL string
		IsAdmin             bool
	}{
		Content: ar.Content,
		Title:   ar.Title,
		URL:     ar.URL,
		//Used because admin-ness was asserted earlier
		IsAdmin: true,
	}

	err = blogTemps.ExecuteTemplate(w, "editor", data)
	fmt.Println(err)
}

func loginRoute(w http.ResponseWriter, r *http.Request) {

	if WebAdmin.AttemptAuth(w, r) {
		http.Redirect(w, r, "/blog/", 303)
	} else {
		buf := new(bytes.Buffer)
		err := blogTemps.ExecuteTemplate(buf, "Login", nil)
		if err != nil {
			internalError(w)
			fmt.Println("Error while displaying login page:", err)
			return
		}
		buf.WriteTo(w)
	}
}

func internalError(w http.ResponseWriter) {
	w.WriteHeader(500)
	w.Write([]byte(Err500.Error()))
}

func getFeed(w http.ResponseWriter, r *http.Request) {
	b := new(bytes.Buffer)
	if err := renderFeed(b); err != nil {
		internalError(w)
		return
	}
	b.WriteTo(w)
}
