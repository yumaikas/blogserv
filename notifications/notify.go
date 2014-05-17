package notifications

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/smtp"
	"time"

	arts "github.com/yumaikas/blogserv/blogArticles"
	"github.com/yumaikas/blogserv/config"
	die "github.com/yumaikas/golang-die"
)

var (
	commentChan = make(chan comment)
	flushChan   = make(chan struct{})
)

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
	commentChan <- submission
}

var emailTemplate *template.Template

func init() {
	var err error
	emailTemplate, err = template.ParseFiles("Templates/email.gohtml")
	//Die on a failed template parse.
	die.OnErr(err)
	go notifyLoop()
	go listenForAdmin()
}

func sendEmail(toNotify []comment) {
	defer func() {
		if die.Log(recover()) != nil {
			//Return comments to the queue
			for _, c := range toNotify {
				commentChan <- c
			}
		}
	}()
	articles := make(map[string][]comment)
	for _, c := range toNotify {
		if ar, found := articles[c.URL]; found {
			ar = append(ar, c)
		} else {
			articles[c.URL] = []comment{c}
		}
	}
	buf := &bytes.Buffer{}
	err := emailTemplate.ExecuteTemplate(buf, "notification", articles)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	auth := config.EmailAuth()
	smtp.SendMail(auth.HostServer, auth.Auth, auth.FromEmail, auth.ToBeNotified, buf.Bytes())
}
func listenForAdmin() {
	l, err := net.Listen("tcp", "localhost:8001")
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
			case "flush":
				flushChan <- struct{}{}
				fmt.Fprintln(c, "flush sent")
			default:
			}
		}
		c.Close()
	}
}

func notifyLoop() {
	toSend := make([]comment, 0)
	//by rate limiting emails to every 15 minutes, we keep from overflowing the gmail 24 hour send limit easily.
	tick := time.Tick(time.Minute * 15)
	for {
		select {
		case <-flushChan:
			if len(toSend) > 0 {
				go sendEmail(toSend)
				toSend = make([]comment, 0)
			}
		case <-tick:
			if len(toSend) > 0 {
				go sendEmail(toSend)
				toSend = make([]comment, 0)
			}
			//Capture a comment for later sending.
		case c := <-commentChan:
			toSend = append(toSend, c)
		}
	}
}
