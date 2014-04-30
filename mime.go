package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

//Here you can write your logging code
func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			url := r.URL
			handler.ServeHTTP(w, r)
		})
}

//Check to see if a string has a suffix
func checkSuffix(url, expected string) bool {
	if len(url) < len(expected) {
		return false
	}
	if url[len(url)-len(expected):] == expected {
		return true
	} else {
		return false
	}
}

var logMux = Log(http.DefaultServeMux)
