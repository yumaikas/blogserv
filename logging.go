// httptest's response recorder is used to stub out the http response so that an error
// doesn't get any exposure to the outside world
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"time"
)

// Here you can write your logging code
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

var logMux = HandlePanicErrs(Log(http.DefaultServeMux))

func HandlePanicErrs(handler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			stubW := httptest.NewRecorder()
			defer func() {
				if errVal := recover(); errVal != nil {
					blogTemps.ExecuteTemplate(w, "serverError", nil)
					fmt.Println("Panic error in server: ", errVal)
					debug.PrintStack()
				} else {
					for key, values := range stubW.HeaderMap {
						for _, value := range values {
							w.Header().Add(key, value)
						}
					}
					w.WriteHeader(stubW.Code)
					stubW.Body.WriteTo(w)
				}
			}()
			handler.ServeHTTP(stubW, r)
		})
}
