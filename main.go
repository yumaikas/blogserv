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
	//If any of these functions fail, we aren't ready for the server to run.
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
func assignPaths() {
	node := http.HandleFunc
	secure := WebAdmin.SecurePath
	authed := WebAdmin.AuthenticatedPath
	//The feed can go over http
	http.HandleFunc("/blog/feed.xml", getFeed)

	//Everything else is either https or https and authenticated as an admin
	node("/", root)
	node("/index.html", secure(home))
	node("/blog", secure(home))
	node("/blog/", secure(getArticle))
	node("/submitComment/", secure(postComment))
	node("/api/", secure(api))
	node("/blog/login", secure(loginRoute))

	//This is secured, and fires auth checks, redirecting to longing if a failure happened
	//node("/admin", secure(adminStart))

	//The entire admin space needs to be authenticated
	node("/admin/create/", authed(create))
	node("/admin/home", authed(adminHome))
	node("/admin/login", authed(performLogin))
	node("/admin/logout", authed(performLogout))
	node("/admin/edit/", authed(edit))
	node("/admin/publish/", authed(publishArticle))
	node("/admin/hideComment/", authed(hideComment))
	node("/admin/showComment/", authed(showComment))
	node("/admin/deleteComment/", authed(deleteComment))
}
func main() {
	//Needs to be called inside main() for some reason. I suspect that it has
	//something to do with initization order, but am not sure.
	addClickOnceMimeTypes()
	assignPaths()

	//For production use port 80
	err := http.ListenAndServe(":6060", logMux)
	//For testing use port 8080
	//err := http.ListenAndServe(":8080", logMux)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func root(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) > 1 {
		//Only serve files to secure paths
		WebAdmin.SecurePath(fileRoot.ServeHTTP)(w, r)
		return
	}
	home(w, r)
}

//Requires auth
func adminHome(w http.ResponseWriter, r *http.Request, userID string) {
	b := new(bytes.Buffer)
	ars, err := AdminArticles()
	fmt.Println(r.Cookies())
	fmt.Println("Admin access")
	for idx, _ := range ars {
		ars[idx].IsAdmin = true
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
		fmt.Printf("An error occurred while rendering templates %s", err.Error())
		w.WriteHeader(500)
		b.WriteTo(w)
		fmt.Fprintf(w, "%s", Err500.Error())
		return
	}

	b.WriteTo(w)
}

func getArticle(w http.ResponseWriter, r *http.Request) {
	_, isAdmin := WebAdmin.AttemptAuth(w, r)
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

//This path needs to be protected at the tree level
func performLogin(w http.ResponseWriter, r *http.Request, userID string) {
	WebAdmin.AddNameCookie(w, r, userID)
	http.Redirect(w, r, "/admin/home", 303)
}

//Requires Auth
func performLogout(w http.ResponseWriter, r *http.Request, userID string) {
	WebAdmin.ClearToken(userID)
	c, err := r.Cookie("authToken")
	if err == nil {
		c.MaxAge = -1
		c.Value = ""
	}
	http.SetCookie(w, c)
	http.Redirect(w, r, "/blog/", 303)
}

//Requires Auth
func create(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method == "GET" {
		err := createSetup(w, r)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintln(w, "Unable to create article.")
			fmt.Fprint(w, Err500.Error())
		}
		return
	}
	if r.Method == "POST" {
		err := createSubmit(w, r)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprint(w, "Unable to save article")
			fmt.Fprint(w, Err500.Error())
		}
		return
	}
	fmt.Println("Invalid method for edit URL.")
	w.WriteHeader(404)
	fmt.Fprint(w, Err404.Error())
}

//This function needs to be auth verified before calling.
func createSetup(w http.ResponseWriter, r *http.Request) error {
	data := struct {
		IsAdmin bool
	}{
		//Used because admin-ness was asserted earlier
		IsAdmin: true,
	}
	buf := &bytes.Buffer{}
	err := blogTemps.ExecuteTemplate(buf, "creator", data)
	if err != nil {
		//Errors are written at the level above this
		return err
	}
	buf.WriteTo(w)
	return err
}

func createSubmit(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	title := r.PostFormValue("Title")
	article := r.PostFormValue("article")
	url := r.PostFormValue("Slug")
	ar := arts.Article{
		Content:      article,
		Title:        title,
		URL:          url,
		PublishStage: "Draft",
	}
	err = arts.SaveArticle(ar)
	if err != nil {
		fmt.Println("Error in saving article", err)
	}
	return err
}

//Requires Auth
func edit(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method == "GET" {
		err := editSetup(w, r)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintln(w, "Unable to edit article.")
			fmt.Fprint(w, Err500.Error())
		}
		return
	}
	if r.Method == "POST" {
		err := editSubmit(w, r)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprint(w, "Unable to save article edits")
			fmt.Fprint(w, Err500.Error())
		}
		return
	}
	fmt.Println("Invalid method for edit URL.")
	w.WriteHeader(404)
	fmt.Fprint(w, Err404.Error())
}

func publishArticle(w http.ResponseWriter, r *http.Request, userID string) {
	articleTitle := r.URL.Path[len("/admin/publish/"):]
	fmt.Println("Attempting to publish article:", articleTitle)
	err := saveArticle(w, r, articleTitle, "Published")
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, "Unable to publish Article")
		fmt.Fprint(w, Err500.Error())
	} else {
		http.Redirect(w, r, "/blog/"+articleTitle, 303)
	}

}

func editSubmit(w http.ResponseWriter, r *http.Request) error {
	articleTitle := r.URL.Path[len("/admin/edit/"):]
	fmt.Println("Attempting to edit article:", articleTitle)
	err := saveArticle(w, r, articleTitle, "Draft")
	if err == nil {
		http.Redirect(w, r, "/admin/edit/"+articleTitle, 303)
	}
	return err
}

func saveArticle(w http.ResponseWriter, r *http.Request, articleTitle, publishStage string) error {
	art, err := fillArticle(articleTitle)
	r.ParseForm()

	title := r.PostFormValue("Title")
	article := r.PostFormValue("article")
	ar := arts.Article(art)
	ar.Content = article
	ar.Title = title
	ar.PublishStage = publishStage
	err = arts.SaveArticle(arts.Article(ar))
	if err != nil {
		fmt.Println("Error in saving article", err)
	}
	return err
}

//This function needs to be auth verified before calling.
func editSetup(w http.ResponseWriter, r *http.Request) error {

	fmt.Println(r.URL.Path)
	articleTitle := r.URL.Path[len("/admin/edit/"):]
	ar, err := fillArticle(articleTitle)

	if err == arts.ErrArticleNotFound {
		fmt.Println("Unable to find article for editing")
		return err
	} else if err != nil {
		return err
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
	return err
}

func loginRoute(w http.ResponseWriter, r *http.Request) {
	if _, isAdmin := WebAdmin.AttemptAuth(w, r); isAdmin {
		http.Redirect(w, r, "/admin/home", 303)
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
