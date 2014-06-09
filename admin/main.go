package main

import (
	"bufio"

	"github.com/yumaikas/blogserv/blogArticles"
	//"bytes"
	"database/sql"
	"flag"
	"fmt"
	"net"
	"net/smtp"

	"github.com/yumaikas/blogserv/config"
	die "github.com/yumaikas/golang-die"

	"code.google.com/p/go.crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"
)

var reset = flag.Bool("reset", false, "reset the templates and config")
var flushComments = flag.Bool("flushComments", false, "Flush the comments from the notification queue. This makes sure that emails are sent")
var password = flag.String("password", "", "Set the admin password for the blog on this computer")
var createUser = flag.Bool("createUser", false, "Create a new user with a password and userID. Requires -userID and -password")
var set = flag.Bool("set", false, "Confirm setting of password.")
var userID = flag.String("userID", "", "the admin for whom to set the password. (Note, don't put this version in the wild)")
var sendTestEmail = flag.Bool("testEmail", false, "Send a test email to make sure that email is working.")
var fillGUIDS = flag.Bool("fillGUIDs", false, "Fill the GUIDS for the comments on the database")
var dumpConfig = flag.Bool("dumpConfig", false, "Show the location of the DB file")

func main() {
	flag.Parse()
	if *reset {
		fmt.Println("Resetting templates")
		sendAdminMessage("reset", "8000")
		return
	}
	if *flushComments {
		fmt.Println("Flushing comments")
		sendAdminMessage("flush", "8001")
		return
	}

	if *createUser && *password != "" && *userID != "" {
		createUserOnDB(*password, *userID)
		return
	}

	if (*password != "" || *userID != "") && !*set {
		fmt.Println("Please use -set to confirm password change")
		return
	}
	if *set && *password != "" && *userID != "" {
		setPassword(*password, *userID)
		return
	}
	if *fillGUIDS {
		fillGUIDComments()
		return
	}
	if *dumpConfig {
		showConfigValues()
		return
	}
	flag.Usage()
}

func showConfigValues() {
	fmt.Print(config.DbPath())
}
func sendAdminMessage(message, port string) {

	c, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		//nothing else we can do
		fmt.Println(message + " failed")
		return
	}
	fmt.Fprintln(c, message)
	buf := bufio.NewScanner(c)
	buf.Scan()
	if err != nil || buf.Text() != message+" sent" {
		fmt.Println("Reset failed")
		return
	}
	fmt.Println("Server told to " + message)
}
func createUserOnDB(password, userID string) {
	defer die.Log("createUserOnDB")

	passBuf := []byte(password)
	hash, err := bcrypt.GenerateFromPassword(passBuf, 13)
	die.OnErr(err)

	db, err := sql.Open("sqlite3", config.DbPath())
	die.OnErr(err)

	_, err = db.Exec("Insert into credentials (password, userID) values (?,?)", hash, userID)
	die.OnErr(err)

	fmt.Println("Password successfully updated.")
}

func setPassword(password, userID string) {
	defer die.Log("setPassword")
	passBuf := []byte(password)
	hash, err := bcrypt.GenerateFromPassword(passBuf, 13)
	die.OnErr(err)

	db, err := sql.Open("sqlite3", config.DbPath())
	die.OnErr(err)

	_, err = db.Exec("Update credentials set password = ? where userID = ?", hash, userID)
	die.OnErr(err)

	fmt.Println("Password successfully updated.")
}

func testEmail() {
	auth := config.EmailAuth()
	fmt.Printf(auth.HostServer)
	//fmt.Printf("%v", auth)
	//fmt.Printf("%v\n", auth.HostServer)
	fmt.Printf("%v", auth.Auth)
	err := smtp.SendMail(auth.HostServer, auth.Auth, auth.FromEmail, auth.ToBeNotified, []byte(
		`Content-Type: text/html
To: yumaikas94@gmail.com
Subject: Test Email	

<html><body>
<b>Test</b> <i>message</i> <del>to</del> confirm email access
</body>
</html>
`))
	if err != nil {
		fmt.Println("Error in testing email!")
		fmt.Println(err.Error())
	}

}

func fillGUIDComments() {
	defer die.Log("filGUIDComments")
	db, err := sql.Open("sqlite3", config.DbPath())
	die.OnErr(err)
	u := `Update Comments set GUID = 'NIL' where GUID is NULL`
	_, err = db.Exec(u)
	die.OnErr(err)
	idsToFill := make([]int, 0)
	q := `Select GUID, id from Comments`
	comments, err := db.Query(q)
	die.OnErr(err)
	for comments.Next() {
		var guid string
		var id int
		err := comments.Scan(&guid, &id)
		die.OnErr(err)
		fmt.Println("Comment guid:", guid)
		if guid == "NIL" {
			idsToFill = append(idsToFill, id)
		}
	}
	_, err = db.Exec(`Update Comments SET GUID = 'NIL' where GUID is NULL`)
	die.OnErr(err)
	fmt.Println(idsToFill)
	tx, err := db.Begin()
	defer func() {
		val := recover()
		if val != nil {
			tx.Rollback()
		}
	}()
	die.OnErr(err)
	updateQuery := `Update Comments SET GUID = ? where id = ?`
	for _, id := range idsToFill {
		guid, err := blogArticles.NewCommentGuid(tx)
		die.OnErr(err)
		_, err = tx.Exec(updateQuery, guid, id)
		die.OnErr(err)
	}
	tx.Commit()
}
