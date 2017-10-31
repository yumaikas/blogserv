// TODO: finish the auth check

package WebAdmin

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/yumaikas/golang-die"
)

func err404ToWriter(w http.ResponseWriter) {
	w.WriteHeader(404)
	w.Write([]byte("Webpage does not exist!"))
}

// Standard web func
type WebFunc func(http.ResponseWriter, *http.Request)

// Ensures that SSL is enabled for a path
func SecurePath(serveRequest WebFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// IsLoopback check for development only. Should be configable to disable.
		if IsLoopback(r) || CheckHTTPS(w, r) {
			serveRequest(w, r)
		} else {
			err404ToWriter(w)
		}
	}
}

// Type for func that must be authenticated. If authentication succeeds, send a user string on the last parameter
type AuthedFunc func(w http.ResponseWriter, r *http.Request, userID string)

// Ensures that the user is authenticated before executing this path
func AuthenticatedPath(protectedFunc AuthedFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if userID, ok := AttemptAuth(w, r); ok {
			protectedFunc(w, r, userID)
		} else {
			err404ToWriter(w)
		}
	}
}

func CheckHTTPS(w http.ResponseWriter, r *http.Request) bool {
	// There are two possibilities
	if r.TLS == nil || !r.TLS.HandshakeComplete {
		fmt.Println("Attempt to use unsecure connection!")
		return false
	}
	return true
}
func IsLoopback(r *http.Request) bool {
	addr := r.RemoteAddr
	isLoopback := strings.HasPrefix(addr, "[::1]") || strings.HasPrefix(addr, "127.0.0.")
	fmt.Println(addr)
	return isLoopback
}

func AddrWithoutPort(r *http.Request) string {
	// Take the string up to the last :, which is right before the port number
	return r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
}

// Use this to track timeout information.
type IPAddress string

var attempts map[IPAddress]int

// This function could be expensive, as it involved either calls to bcrypt or a time limiter
func AttemptAuth(w http.ResponseWriter, r *http.Request) (userID string, validAuth bool) {
	// There are two possibilities, one that we have an auth cookie already,
	// or that we have an attempt to username/password verify

	if err := r.ParseForm(); err != nil {
		return "", false
	}

	defer func() {
		val := recover()
		if val != nil {
			fmt.Println("Error in AuthAttempt", val)
			fmt.Println(val)
			userID = ""
			validAuth = false
			sleepForBadRequest()
		}
	}()

	pass, name := r.FormValue("password"), r.FormValue("userName")
	// If either value is empty
	if !(pass == "" || name == "") {
		if !checkLoginCreds(pass, name, r.RemoteAddr) {
			fmt.Println("Sleeping to rate limit bad requests, and to keep from DOS attacks")
			sleepForBadRequest()
			return "", false
		}
		token, err := GenerateRandomString()
		fmt.Println("Attempted to generate auth token")
		die.OnErr(err)
		fmt.Println("Generated auth token")
		err = setAuthToken(token, name, AddrWithoutPort(r))
		die.OnErr(err)
		fmt.Println("Saved auth token")
		return name, true
	}

	// Attempting cookie based authentication. Need to put a sleep of some sort in here...
	c, err := r.Cookie("authToken")
	if c != nil {
		fmt.Print("Cookie:", c.Raw, "")
	}
	if err != nil {
		fmt.Println("Error (if any)", err)
		return "", false
	}
	if err == nil && c.Value != "" {
		// Get the authToken from the database
		userID, err = idFromTokenAndIP(c.Value, AddrWithoutPort(r))
		if err != nil {
			fmt.Println("Error in checking database for authentication token" + err.Error())
			sleepForBadRequest()
			return "", false
		}
		fmt.Println(c)
		return userID, true
	}
	fmt.Println("Error in logging in:", err)
	return "", false

}
func sleepForBadRequest() {
	sleepSecond, err := rand.Int(rand.Reader, big.NewInt(1000))
	// If we can't get any randomness, recover by using 5000 milliseconds, so that we at least get rate limiting
	if err != nil {
		sleepSecond = big.NewInt(5000)
	}
	// On failure to authenticate, sleep between .5 to 10 seconds. HAHAHAHAHAHAHA
	time.Sleep(time.Millisecond*time.Duration(sleepSecond.Int64()) + 500)
}

func AddNameCookie(w http.ResponseWriter, r *http.Request, userID string) {
	token, err := tokenFromIDandIPAddr(userID, AddrWithoutPort(r))
	if err != nil {
		fmt.Println("Error fetching token from DB" + err.Error())
		return
	}
	setAuthCookie(w, r, token)
}

func setAuthCookie(w http.ResponseWriter, r *http.Request, token string) {
	domain := "localhost"
	if !IsLoopback(r) {
		// TODO pull from config
		domain = ".junglecoder.com"
	}
	cookie := &http.Cookie{
		// When testing on localhost
		Domain:   domain,
		Value:    token,
		HttpOnly: true,
		Expires:  time.Now().AddDate(0, 0, 12),
		// Security of the cookie depends on who
		Secure: !IsLoopback(r),
		Path:   "/",
		Name:   "authToken",
	}
	http.SetCookie(w, cookie)
}
func GenerateRandomString() (token string, err error) {
	var b [16]byte
	num, err := rand.Read(b[:])
	if num != 16 || err != nil {
		return "", err
	}
	uuid := fmt.Sprintf("%x%x%x%x%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid, err
}
