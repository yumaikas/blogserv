//The code in here is panicky, and uses defer and recover to handle errors

package WebAdmin

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/yumaikas/golang-die"

	"code.google.com/p/go.crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/yumaikas/blogserv/config"
)

func ClearToken(userID string) {
	//Blanking IP address and token
	setAuthToken("", userID, "")
}

func idFromTokenAndIP(token, IPAddr string) (userID string, retErr error) {
	db, err := dbOpen()
	defer db.Close()
	if err != nil {
		return "", err
	}
	q := "Select UserID from AuthToken where Token = ? and IPAddress = ? limit 1"
	err = db.QueryRow(q, token, IPAddr).Scan(&userID)
	if err != nil {
		return "", err
	}
	//Returning named parameters.
	return userID, nil
}

func tokenFromIDandIPAddr(userID, IPAddr string) (token string, retErr error) {
	db, err := dbOpen()
	defer db.Close()
	//The token is never set, and so it is "" if this fails
	die.OnErr(err)

	q := "Select token from authToken where UserID = ? and IPAddress = ? limit 1"
	if err = db.QueryRow(q, userID, IPAddr).Scan(&token); err != nil {
		return "", err
	}
	//Returning named parameters.
	return
}

func setAuthToken(token, userID, IPAddr string) (retErr error) {
	defer func() {
		if val := recover(); val != nil {
			retErr = fmt.Errorf("Error in setting auth token: %v", val)
		}
	}()
	db, err := dbOpen()
	defer db.Close()
	die.OnErr(err)

	res, err := db.Exec("Update authToken set token = ?, IPAddress = ? where userID = ?", token, IPAddr, userID)
	die.OnErr(err)
	if num, e := res.RowsAffected(); num > 1 || e != nil {
		die.OnErr(e)
		//Otherwise, complain about the wrong number of rows being updated.
		die.OnErr(errors.New(fmt.Sprint("Wrong number of rows for user", userID, ".Please check integrity of database.")))
	} else if num == 0 {
		//As this is an insert, the number of rows affected shouldn't matter, since it should always be one.
		//The main cause for it not to be one would be some kind of schema error, which would be captured into err
		_, err = db.Exec("Insert into authToken(userID, token, IPAddress) values (?,?,?) ", userID, token, IPAddr)
		die.OnErr(err)
	}
	return
}

func checkLoginCreds(password, userName, remoteAddr string) (canLogin bool) {
	var err error
	defer die.LogSettingReturns("checkLoginCreds", &err, func() { canLogin = false })

	db, err := dbOpen()
	defer db.Close()
	die.OnErr(err)

	fmt.Println("poiu")

	var dbPass string
	q := "Select password from credentials where userID = ?"
	die.OnErr(db.QueryRow(q, userName).Scan(&dbPass))

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
		die.OnErr(err)
	}
	fmt.Println("Login Creds succeeded!")
	return true
}

func dbOpen() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", config.DbPath())
	return db, err
}

func valToErr(val interface{}) error {
	switch val.(type) {
	case nil:
		return nil
	case error:
		return val.(error)
	default:
		return fmt.Errorf("%v", val)
	}
}
