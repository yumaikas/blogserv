//The code in here is panicky, and uses defer and recover to handle errors

package WebAdmin

import (
	"database/sql"
	"errors"
	"fmt"

	"code.google.com/p/go.crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/yumaikas/blogserv/config"
)

//Fixing in general: Add code to create a record for a user if said user doesn't exist

//FIXME: get rid of hard coded string
func ClearToken(userID string) {
	setAuthToken("", "yumaikas")
}

func authToken(userID string) (token string, retErr error) {
	defer func() {
		if recov(&retErr) {
			token = ""
		}
	}()

	db, err := dbOpen()
	defer db.Close()
	dieOnErr(err)

	dieOnErr(db.QueryRow("Select token from authToken limit 1").Scan(&token))
	//Returning named parameters.
	return
}

func setAuthToken(token, userID string) (retErr error) {
	defer recov(&retErr)
	db, err := dbOpen()
	defer db.Close()
	dieOnErr(err)

	res, err := db.Exec("Update authToken set token = ? where userID = ?", token, userID)
	dieOnErr(err)

	if num, e := res.RowsAffected(); num != 1 {
		//if an error is still here, throw that first.
		dieOnErr(e)
		//Otherwise, complain about the wrong number of rows being updated.
		dieOnErr(errors.New("Wrong number of rows in auth table, please check the integrity of the database"))
	}
	return
}

//This funciton and the one following it are a unit of work
func checkLoginCreds(password, userName, remoteAddr string) (canLogin bool) {
	defer func() {
		//logErr calls recover()
		if logErr("Error in login code") {
			canLogin = false
		}
	}()

	db, err := dbOpen()
	defer db.Close()
	dieOnErr(err)

	var dbPass string
	query := "Select password from credentials where userID = ?"
	fmt.Println("poiu")
	dieOnErr(db.QueryRow(query, userName).Scan(&dbPass))
	fmt.Println("asdf")

	//Cast the types as needed
	dbBuf := []byte(dbPass)
	localBuf := []byte(password)

	err = nil
	err = bcrypt.CompareHashAndPassword(dbBuf, localBuf)
	if err == bcrypt.ErrMismatchedHashAndPassword {
		fmt.Println("Invalid attempt to login from ISP", remoteAddr)
		return false
	} else {
		//Code error here
		dieOnErr(err)
	}
	fmt.Println("yyyyyyyyyyyyyyyyyy")
	return true
}

func dbOpen() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", config.DbPath())
	return db, err
}

func dieOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
func recov(err *error) bool {
	val := recover()
	switch val.(type) {
	case nil:
		return false
	case error:
		*err = val.(error)
	default:
		*err = fmt.Errorf("%v", val)
	}
	return true
}
func logErr(preamble string) bool {
	val := recover()
	if val == nil {
		return false
	}
	if val != nil && preamble == "" {
		fmt.Println(val)
	} else {
		fmt.Println(preamble, val)
	}
	return true
}

//func setAuthToken(token string) (retErr error) {
//}
