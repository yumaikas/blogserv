package main

import (
	"fmt"
	"net/http"
	"time"
)

//Here you can write your logging code
func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("%s:", time.Now().Format("2006-01-02-15:04:05"))
			fmt.Printf("Request:{ '%s' '%s' '%s' } Agent: { %s }\n",
				r.RemoteAddr, r.Method, r.URL,
				r.UserAgent())
			handler.ServeHTTP(w, r)
		})
}

var logMux = Log(http.DefaultServeMux)
