// The code in here is panicky, and uses defer and recover to handle errors

package WebAdmin

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/yumaikas/golang-die"

	"code.google.com/p/go.crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/yumaikas/blogserv/config"
)

func ClearToken(userID string) {
	// Blanking IP address and token
	setAuthToken("", userID, "")
}

var tokenExpired = errors.New("Token expired")

func idFromTokenAndIP(token, IPAddr string) (userID string, retErr error) {
	fmt.Println("HEXAGOAL:", token, IPAddr)
	db, err := dbOpen()
	defer db.Close()
	if err != nil {
		return "", err
	}
	var expiration string
	// q := "Select UserID, Expiration from AuthToken where Token = ? and IPAddress = ? limit 1"
	q := "Select UserID, Expiration from AuthToken where Token = ? limit 1"
	die.OnErr(db.QueryRow(q, token/*, IPAddr*/).Scan(&userID, &expiration))
	t, terr := time.Parse(time.RFC3339, expiration)
	if terr != nil {
		return "", terr
	} else if err != nil {
		return "", err
	} else if t.Before(time.Now()) {
		return "", tokenExpired
	}
	// Returning named parameters.
	return userID, nil
}

func tokenFromIDandIPAddr(userID, IPAddr string) (token string, retErr error) {
	db, err := dbOpen()
	defer db.Close()
	// The token is never set, and so it is "" if this fails
	die.OnErr(err)

	var expiration string
	q := "Select token, Expiration from authToken where UserID = ? limit 1"
	die.OnErr(db.QueryRow(q, userID).Scan(&token, &expiration))
	t, terr := time.Parse(time.RFC3339, expiration)
	if terr != nil {
		return "", err
	} else if err != nil {
		return "", terr
	} else if t.Before(time.Now()) {
		return "", tokenExpired
	}
	// Returning named parameters.
	return
}

func setAuthToken(token, userID, IPAddr string) (retErr error) {
	db, err := dbOpen()
	defer db.Close()
	die.OnErr(err)
	expiration, err := time.Now().Add(2 * time.Hour).MarshalText()
	res, err := db.Exec("Update authToken set token = ?, IPAddress = ?, Expiration = ? where userID = ?",
		token, IPAddr, string(expiration),
		userID)
	if err != nil {
		return err
	}
	if num, e := res.RowsAffected(); num > 1 || e != nil {
		die.OnErr(e)
		// Otherwise, complain about the wrong number of rows being updated.
		die.OnErr(errors.New(fmt.Sprint("Wrong number of rows for user", userID, ".Please check integrity of database.")))
	} else if num == 0 {
		// As this is an insert, the number of rows affected shouldn't matter, since it should always be one.
		// The main cause for it not to be one would be some kind of schema error, which would be captured into err
		_, err = db.Exec("Insert into authToken(userID, token, IPAddress) values (?,?,?) ", userID, token, IPAddr)
		die.OnErr(err)
	}
	return
}

func checkLoginCreds(password, userName, remoteAddr string) (canLogin bool) {
	var err error
	canLogin = false

	db, err := dbOpen()
	defer db.Close()
	die.OnErr(err)

	fmt.Println("poiu")

	var dbPass string
	q := "Select password from credentials where userID = ?"
	die.OnErr(db.QueryRow(q, userName).Scan(&dbPass))

	fmt.Println("asdf")

	// Cast the types as needed
	dbBuf := []byte(dbPass)
	localBuf := []byte(password)

	err = nil
	err = bcrypt.CompareHashAndPassword(dbBuf, localBuf)
	if err == bcrypt.ErrMismatchedHashAndPassword {
		fmt.Println("Invalid attempt to login from ISP", remoteAddr)
		return false
	} else {
		// Code error here
		die.OnErr(err)
	}
	fmt.Println("Login Creds succeeded!")
	return true
}

func dbOpen() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", config.DbPath())
	return db, err
}
