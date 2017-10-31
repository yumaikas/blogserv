package main // blogservModels

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/yumaikas/blogserv/WebAdmin"
	"github.com/yumaikas/blogserv/notifications"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tgascoigne/akismet"
	arts "github.com/yumaikas/blogserv/blogArticles"
	"github.com/yumaikas/blogserv/config"
)

var (
	Err404 error           = errors.New("404: The article you are looking for doesn't exist.")
	Err500 error           = errors.New("500: Something went wrong in the server. :(")
	akis   *akismet.Config = &akismet.Config{
		APIKey:    config.AkismetKey(),
		Host:      "http:// www.yumaikas.com",
		UserAgent: akismet.UserAgentString("blogserv/0.5.1"),
	}
	// db *sql.DB
)

type Article arts.Article
type Comment arts.Comment

// Handy for debugging things
func dump(me string) string {
	fmt.Println(me)
	return me
}

func akismet_init() {
	err := akismet.VerifyKey(akis)
	if err != nil {
		log.Fatal("Invalid akismet api key")
	}
}

func RSSArticles() (rssfeed, error) {
	ars, err := listArticles(arts.IsPublished)
	return rssfeed(ars), err
}
func listArticles(filter func(arts.Article) bool) (articleList, error) {
	ars, err := arts.ListArticles()
	artemp := make([]arts.Article, 0)
	for _, val := range ars {
		if filter(val) {
			artemp = append(artemp, val)
		}
	}
	return articleList(artemp), err
}
func AdminArticles() (articleList, error) {
	return arts.ListArticles()
}
func DraftsArticles() (articleList, error) {
	return listArticles(arts.IsDraft)
}

func HTMLArticles() (articleList, error) {
	return listArticles(arts.IsPublished)
}

// Populates an article based on a title.
func fillArticle(URL string) (Article, error) {
	ar, err := arts.FillArticle(URL)
	if err == arts.ErrArticleNotFound {
		fmt.Println(err)
		return Article(ar), err
	} else if err != nil {
		fmt.Println(err)
		return Article(ar), Err500
	}
	return Article(ar), nil
}

func (ars articleList) IsAdmin() bool {
	for _, ar := range ars {
		if !ar.IsAdmin {
			return false
		}
	}
	return true
}

func postComment(w http.ResponseWriter, r *http.Request) {
	articleName := r.URL.Path[len("/submitComment/"):]

	redirect := func() {
		url := "/blog/" + articleName
		http.Redirect(w, r, url, 303)
	}
	r.ParseForm()
	comment := akismet.Comment{
		UserIP:      r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		Author:      r.FormValue("author"),
		AuthorEmail: r.FormValue("email"),
		Content:     r.FormValue("Comment"),
	}
	err := akismet.CommentCheck(akis, comment)
	if err != nil {
		switch err {
		case akismet.ErrSpam:
			arts.SpamToDB(comment, articleName)
			return
		case akismet.ErrInvalidRequest:
			log.Printf("Aksimet request invalid: %s\n", err.Error())
		case akismet.ErrUnknown:
			log.Printf("An abnormal error happened when querying akismet: %s", err.Error())
		case akismet.ErrInvalidKey:
			log.Printf("Aksimet key invalid, no spam checking available.")
		}
		return // Nothing more we can do here for now
	}

	// Notify here. send the request in.
	err = arts.CommentToDB(comment, articleName)
	if err != nil {
		fmt.Println("Error when submitting comment: ", err)
		http.Error(w, err.Error(), 500)
		return
	}
	// Add a successful comment to the notification
	notifications.NotifyComment(arts.Comment{UserName: comment.Author, Content: comment.Content},
		comment.AuthorEmail,
		articleName,
		articleName,
		r)
	redirect()
}

// Made to only redirect when the referer is from the website. I don't want a open redirect relay
var (
	loopbackReferer = regexp.MustCompile(`^http://localhost:\d*/blog/(.*)`)
	// TODO: localize this to pull host value from config
	productionReferer = regexp.MustCompile(`^https://(www)?\.junglecoder\.com:(\d+)?/blog/(.*)`)
)

var showComment = adminComment("/admin/showComment/", arts.ShowComment)
var hideComment = adminComment("/admin/hideComment/", arts.HideComment)
var deleteComment = adminComment("/admin/deleteComment/", arts.DeleteComment)

func listComments(w http.ResponseWriter, r *http.Request, userID string) {
	comments, err := arts.ListAllComments()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = renderCommentsAdmin(comments, w)
	if err != nil {
		log.Fatal(err.Error())
	}
}

// This is a template for comment administration
func adminComment(path string, adminAction func(string) error) WebAdmin.AuthedFunc {
	return func(w http.ResponseWriter, r *http.Request, userID string) {
		guid := r.URL.Path[len(path):]
		articlePath, err := arts.ArticleFromComment(guid)
		if err != nil {
			fmt.Println("In path:", path, "Error occured:", err)
			w.WriteHeader(500)
			fmt.Fprint(w, Err500.Error())
			return
		}
		err = adminAction(guid)
		if err != nil {
			fmt.Println("In path:", path, "Error occured:", err)
			w.WriteHeader(500)
			fmt.Fprint(w, Err500.Error())
			return
		}
		fmt.Println("Referer:", r.Referer())
		if WebAdmin.IsLoopback(r) && loopbackReferer.MatchString(r.Referer()) {
			fmt.Println("Redirecting on loopback")
			http.Redirect(w, r, r.Referer(), 303)
			return
		} else if productionReferer.MatchString(r.Referer()) {
			fmt.Println("Redirecting on production")
			http.Redirect(w, r, r.Referer(), 303)
			return
		}
		http.Redirect(w, r, "/blog/"+articlePath, 303)
	}
}

// These are the values that are populated in the comment.
/*
	    UserIP:      r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		Author:      r.FormValue("author"),
		AuthorEmail: r.FormValue("email"),
		Content:     r.FormValue("Comment"),
*/

type queryComment struct {
	Sql  string
	Args func() (int, int, string)
}
