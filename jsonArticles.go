package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	arts "github.com/yumaikas/blogserv/blogArticles"
)

type jsonArticle struct {
	Title   string "Title"
	Slug    string "Slug"
	Content string "Content"
}

func api(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		fmt.Println(r)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ars, err1 := jsonArticles()
	buf, err := json.Marshal(ars)
	if err != nil {
		//Render error straight to buffer. Debug only.
		fmt.Fprintln(w, err1)
		fmt.Fprint(w, err)
		return
	}
	fmt.Fprint(w, string(buf))
	//TODO: implemnt a rest-ful API here. Current
	//Get the method to dispacth to.
	//a := r.URL.Path[len("/api/"):]
}

func jsonArticles() ([]jsonArticle, error) {
	dbArs, err := listArticles(arts.IsPublished)
	if err != nil {
		fmt.Println("Error loading json articles: " + err.Error())
		return nil, Err500
	}
	ars := make([]jsonArticle, 0)
	for _, ar := range dbArs {
		ars = append(ars, jsonArticle{
			Title:   ar.Title,
			Slug:    ar.URL,
			Content: ar.Content,
		})
	}

	if len(ars) == 0 {
		return nil, Err404
	}
	return ars, nil
}
