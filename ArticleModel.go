package main //blogservModels

import (
	secret "blogserv_secret"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tgascoigne/akismet"
	"log"
	"net/http"
)

type Comment struct {
	UserName, Content string
}

var (
	Err404 error           = errors.New("404: The article you are looking for doesn't exist.")
	Err500 error           = errors.New("500: Something went wrong in the server. :(")
	config *akismet.Config = &akismet.Config{
		APIKey:    secret.AkismetKey,
		Host:      "http://www.yumaikas.com",
		UserAgent: akismet.UserAgentString("blogserv/0.5.0"),
	}
	//db *sql.DB
)

//Hand ownership of the database handle to the calling method
func dbOpen() (*sql.DB, func(), error) {
	db, err := sql.Open("sqlite3", secret.DBPath)
	drop := func() {
		db.Close()
	}
	return db, drop, err
}
func akismet_init() {
	err := akismet.VerifyKey(config)
	if err != nil {
		log.Fatal("Invalid akismet api key")
	}
}

func listArticles() ([]Article, error) {
	var ars = make([]Article, 0)
	db, drop, err := dbOpen()
	defer drop()
	if err != nil {
		return nil, Err500
	}

	//The article query
	rows, err := db.Query(`
	Select Title, URL, Content 
	from Articles Order by id Desc`)
	if err != nil {
		return nil, Err500
	}
	for rows.Next() {
		var ar Article
		rows.Scan(&ar.Title, &ar.URL, &ar.content)
		ars = append(ars, ar)
	}
	if err != nil {
		//Log to output, but simply throw a 500 error to the user
		fmt.Printf("An error occured while trying to fetch articles: %s", err.Error())
		return ars, Err500
	}
	return ars, nil
}

func RSSArticles() (rssfeed, error) {
	ars, err := listArticles()
	return rssfeed(ars), err
}
func HTMLArticles() (articleList, error) {
	ars, err := listArticles()
	return articleList(ars), err
}

//Populates an article based on a title.
func fillArticle(URL string) (Article, error) {
	var ar Article
	db, drop, err := dbOpen()
	defer drop()
	if err != nil {
		return ar, Err500
	}

	var articleId int
	err = db.QueryRow(`
		Select Title, URL, Content, id 
		from Articles 
		where URL = ?`,
		URL).Scan(&ar.Title, &ar.URL, &ar.content, &articleId)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return ar, Err404
		default:
			//debug, for production use fmt.PrintF(err)
			log.Fatal(err)
			return ar, err
		}
	}
	//Get the comments for the article
	commentQ, err := db.Prepare(`
	Select U.screenName, C.Content from 
	Comments as C
	inner join Users as U on C.UserID = U.id
	where C.ArticleID = ?`)
	if err != nil {
		//debug, for production use fmt.PrintF(err)
		log.Fatal(err)
		return ar, err
	}

	rows, err := commentQ.Query(articleId)
	if err != nil {
		//debug, for production use fmt.PrintF(err)
		log.Fatal(err)
		return ar, err
	}

	ar.Comments = make([]Comment, 0)

	for rows.Next() {
		var c Comment
		err = rows.Scan(&c.UserName, &c.Content)
		if err != nil {
			//debug, for production use fmt.Printf(err)
			log.Fatal(err)
			return ar, err
		}

		if len(c.Content) > 0 {
			ar.Comments = append(ar.Comments, c)
		}
	}
	if len(ar.Comments) == 0 {
		ar.Comments = nil
	}
	return ar, nil
}

func postComment(w http.ResponseWriter, r *http.Request) {
	articleName := r.URL.Path[len("/submitComment/"):]

	r.ParseForm()
	comment := akismet.Comment{
		UserIP:      r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		Author:      r.FormValue("author"),
		AuthorEmail: r.FormValue("email"),
		Content:     r.FormValue("Comment"),
	}
	err := akismet.CommentCheck(config, comment)
	if err != nil {
		switch err {
		case akismet.ErrSpam:
			SpamToDB(comment, articleName)
			return
		case akismet.ErrInvalidRequest:
			log.Printf("Aksimet request invalid: %s\n", err.Error())
		case akismet.ErrUnknown:
			log.Printf("An abnormal error happened when querying akismet: %s", err.Error())
		case akismet.ErrInvalidKey:
			log.Printf("Aksimet key invalid, no spam checking available.")
		}
		return //Nothing more we can do here for now
	}
	err = CommentToDB(comment, articleName)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	url := "/blog/" + articleName
	http.Redirect(w, r, url, 303)
}

//Add code to check for the user, and insert the user if need be
//These are the values that are populated in the comment.
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

//Currently do nothing
func SpamToDB(c akismet.Comment, arName string) error {
	return nil
}

func CommentToDB(c akismet.Comment, arName string) error {
	fmt.Print("Enter CommentToDB")
	defer fmt.Print("Exit CommentToDB")
	db, drop, err := dbOpen()
	defer drop()
	if err != nil {
		return Err500
	}

	tx, err := db.Begin()
	rb := func(e error) error {
		tx.Rollback()
		return e
	}
	if err != nil {
		return rb(err)
	}

	//This is what is going in to the db
	in := struct {
		UserID, ArticleID int
		Content           string
	}{0, 0, c.Content}

	err = tx.QueryRow(`Select id from Users where Email = ?`, c.AuthorEmail).Scan(&in.UserID)
	switch {
	case err == sql.ErrNoRows:
		var u_err error
		in.UserID, u_err = addUser(c, tx)
		if u_err != nil {
			return rb(err)
		}
		break
	case err != nil:
		return rb(err)
	}
	err = tx.QueryRow(`Select id from Articles where URL = ?`, arName).Scan(&in.ArticleID)
	if err != nil {
		return rb(err)
	}
	//The results and error(if any)
	r, err := tx.Exec(`
	Insert into Comments (UserID, ArticleID, Content) 
	values (?, ?, ?)`,
		in.UserID, in.ArticleID, in.Content)
	if err != nil {
		return rb(err)
	}
	numRows, err := r.RowsAffected()
	switch {
	case err != nil:
		return rb(err)
	case numRows != 1:
		return rb(fmt.Errorf("Error: %d rows were affected instead of 1", numRows))
	}
	err = tx.Commit()
	if err != nil {
		return rb(err)
	}
	return nil
}

func addUser(c akismet.Comment, tx *sql.Tx) (int, error) {
	//fmt.Print("Enter addUser")
	//defer fmt.Print("Exit addUser")
	r, err := tx.Exec("Insert into Users (screenName, Email) values (?, ?)",
		c.Author, c.AuthorEmail)
	if err != nil {
		return 0, err
	}
	cnt, err := r.RowsAffected()
	switch {
	case err != nil:
		return 0, err
	case cnt != 1:
		return 0, fmt.Errorf("%d rows affected instead of 1", cnt)
	}
	//Return the new userID
	if id, err := r.LastInsertId(); err == nil {
		return int(id), nil
	}
	return 0, err
}
