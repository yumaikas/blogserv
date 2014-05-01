package notifications

import (
	_ "fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"time"

	arts "github.com/yumaikas/blogserv/blogArticles"
	"github.com/yumaikas/blogserv/config"
	die "github.com/yumaikas/golang-die"
)

var comments = make(chan comment)

type comment struct {
	arts.Comment
	Email, IPaddr, URL, ArticleName string
}

type emailEntry struct {
	ArticleName, URL string
	Comments         []comment
}

func NotifyComment(c arts.Comment, email, URL, ArtName string, r *http.Request) {
	submission := comment{c, email, r.RemoteAddr, r.URL.RequestURI(), ArtName}
	comments <- submission
}

func init() {
	temp, err := template.ParseFiles("email.gohtml")
	//Die on a failed template parse.
	die.OnErr(err)
	go notifyLoop()
}

func sendEmail(comments []comment) {
	articles := make(map[string][]comment)
	for _, c := range comment {
		if ar, found := articles[c.URL]; found {
			ar = append(ar, c)
		} else {
			articles[c.URL] = emailEntry{
				c.ArticleName,
				c.URL,
				make([]comment, 0),
			}
			articles.Comments = append(articles.Comments, c)
		}
	}
	auth, host := config.EmailAuth()
	smtp.SendMail(host, config.EmailAuth(), "slaveofyumaikas@gmail.com", to, []byte("Test Email"))
}

func notifyLoop() {
	toSend := make([]comment, 0)
	//by rate limiting emails to every 15 minutes, we keep from overflowing the gmail 24 hour send limit easily.
	tick := time.Tick(time.Minute * 15)
	for {
		select {
		case <-tick:
			go sendEmail(toSend)
			toSend = make([]comment, 0)
			//Capture a comment for later sending.
		case c := <-comments:
			toSend = append(toSend, c)
		}
	}
}
