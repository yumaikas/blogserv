package main

import (
	"bufio"
	//"bytes"
	"database/sql"
	"flag"
	"fmt"
	"net"
	"net/smtp"
	"yumaikas/blogserv/config"
	"yumaikas/die"

	"code.google.com/p/go.crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"
)

var reset = flag.Bool("reset", false, "reset the templates and config")
var password = flag.String("password", "", "Set the admin password for the blog on this computer")
var set = flag.Bool("set", false, "Confirm setting of password.")
var userID = flag.String("userID", "", "the admin for whom to set the password. (Note, don't put this version in the wild)")
var sendTestEmail = flag.Bool("testEmail", false, "Send a test email to make sure that email is working.")

func main() {
	flag.Parse()
	if *reset {
		fmt.Println("Resetting")
		sendReset()
	}
	if (*password != "" || *userID != "") && !*set {
		fmt.Println("Please use -set to confirm password change")
	}
	if *set && *password != "" && *userID != "" {
		setPassword(*password, *userID)
	}
	//	if *sendTestEmail {
	testEmail()
	//	}
}

func sendReset() {
	c, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		//nothing else we can do
		fmt.Println("Reset failed")
		return
	}
	fmt.Fprintln(c, "reset")
	buf := bufio.NewScanner(c)
	buf.Scan()
	if err != nil || buf.Text() != "Reset sent" {
		fmt.Println("Reset failed")
		return
	}
	fmt.Println("Server told to reset")
}

func setPassword(password, userID string) {
	defer die.Log()

	passBuf := []byte(password)
	hash, err := bcrypt.GenerateFromPassword(passBuf, 13)
	die.OnErr(err)

	db, err := sql.Open("sqlite3", config.DbPath())
	die.OnErr(err)

	_, err = db.Exec("Update credentials set password = ? where userID = ?", hash, userID)
	die.OnErr(err)

	fmt.Println("Password successfully updated.")
}

func dieOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func logErr() interface{} {
	val := recover()
	if val != nil {
		fmt.Println(val)
		return val
	}
	return nil
}

func recovErr(err *error) {
	*err = recover().(error)
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
