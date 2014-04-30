package main

import (
	"database/sql"
	"flag"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	secret "github.com/yumaikas/blogserv/config"
)

var test = flag.Bool("test", false, "if set, only display the actions that were going to be taken")

func dbOpen() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", secret.DbPath())
	if err != nil {
		return nil, err
	}
	return db, nil
}
func getArticles() (map[string]*article, error) {
	db, err := sql.Open("sqlite3", secret.DbPath())
	defer db.Close()
	if err != nil {
		return nil, err
	}
	articles := make(map[string]*article)
	arts, err := db.Query(`Select Title, URL, content from Articles`)
	if err != nil {
		return nil, err
	}
	for arts.Next() {
		ar := new(article)
		err = arts.Scan(&ar.Title, &ar.URL, &ar.Content)
		if err != nil {
			return nil, err
		}
		articles[ar.URL] = ar
	}
	return articles, nil
}

func update(a article) {
	if *test {
		return
	}
	db, err := dbOpen()
	tx, err := db.Begin()
	if err != nil {
		panic("DB open failed")
	}
	defer db.Close()
	res, err := tx.Exec(`
	Update Articles
	set Title = ?, Content = ?
	where URL = ?
	`, a.Title, a.Content, a.URL)
	cnt, err1 := res.RowsAffected()
	if cnt > 1 || err1 != nil || err != nil {
		tx.Rollback()
		panic(fmt.Sprintf("Update for %s Failed. %v rows would have been affected", a.URL, cnt))
	}
	tx.Commit()
}
func insert(a article) {
	if *test {
		return
	}
	db, err := dbOpen()
	tx, err := db.Begin()
	if err != nil {
		panic("DB open failed")
	}
	defer db.Close()
	fmt.Println(a.Title)
	fmt.Println(a.URL)
	res, err := tx.Exec(`
	Insert into Articles(Title, Content, URL) 
	values (?, ?, ?)
	`, a.Title, a.Content, a.URL)
	if err != nil {
		panic(err)
	}
	cnt, err1 := res.RowsAffected()
	if cnt > 1 || err1 != nil || err != nil {
		tx.Rollback()
		panic(fmt.Sprintf("Insert for %s Failed. %v rows would have been affected", a.URL, cnt))
	}
	tx.Commit()
}
func (a *article) String() string {
	return a.Title
}
