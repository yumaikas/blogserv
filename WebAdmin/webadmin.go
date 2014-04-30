//TODO: finish the auth check

package WebAdmin

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"
)

func CheckHTTPS(w http.ResponseWriter, r *http.Request) bool {
	//There are two possibilities
	if r.TLS == nil || !r.TLS.HandshakeComplete {
		fmt.Println("Attempt to use unsecure connection!")
		return false
	}
	return true
}

//This function could be expensive, as it involved either calls to bcrypt or
func AttemptAuth(w http.ResponseWriter, r *http.Request) (validAuth bool) {
	//There are two possibilities, one that we have an auth cookie already,
	//or that we have an attempt to username/password verify
	//Code below elided for localhost checking
	r.ParseForm()
	if !CheckHTTPS(w, r) {
		fmt.Println("Attempt to connect over insecure connection. If found on production, stop server imdmediatelly")
		//return false
	}

	if err := r.ParseForm(); err != nil {
		return false
	}
	c, err := r.Cookie("authToken")
	//Attempting cookie based authentication.
	if err == nil && c.Value != "" {
		//Get the authToken from the database
		fmt.Println(";lab")
		auth, err := authToken("yumaikas")
		if err != nil {
			fmt.Println("Error in checking database for authentication token" + err.Error())
			return false
		}
		//Check the auth and expiration.
		if auth == c.Value /*&& c.Expires.After(time.Now()) && c.HttpOnly == true*/ {
			fmt.Println("Need to set SecureOnly when running produciton")
			return true
		}
	}
	fmt.Println("yqe;")
	defer func() {
		val := recover()
		if val != nil {
			fmt.Println(val)
			validAuth = false
		}
	}()

	pass, name := r.FormValue("password"), r.FormValue("userName")
	//If either value is empty
	fmt.Println(pass, name)
	if pass == "" || name == "" {
		return false
	}
	if !checkLoginCreds(pass, name, r.RemoteAddr) {
		fmt.Println("uuuuuuuuuuuuuuuuuuuuu")
		return false
	}
	token, err := generateAuthToken()
	dieOnErr(err)
	err = setAuthToken(token, name)
	dieOnErr(err)
	return true
}

func AddNameCookie(w http.ResponseWriter, r *http.Request) {
	defer logErr("Error fetching token from db")
	token, err := authToken("yumaikas")
	dieOnErr(err)
	setAuthCookie(w, token)
}

func setAuthCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		//When testing on localhost
		//Domain:   ".localhost",
		Value:    token,
		HttpOnly: true,
		Expires:  time.Now().AddDate(0, 0, 12),
		//Secure:   true,
		Path: "/",
		Name: "authToken",
	}
	http.SetCookie(w, cookie)
}
func generateAuthToken() (token string, err error) {
	var b [16]byte
	num, err := rand.Read(b[:])
	if num != 16 || err != nil {
		return "", err
	}
	uuid := fmt.Sprintf("%x%x%x%x%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid, err
}
