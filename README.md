# Blogserv - Work In Progress blog engine in golang.

This is the software that powers https://junglecoder.com. It is currently only intended for personal use by those who are interested in go. It runs on both windows and linux with only changing an environment variable for configurations. 

Just added:

### Web administration features
* Editing articles (with many thanks to epiceditor)
* Creating New articles 
* Makeing sure that IP addresses are part of the login scheme
* Administering comments
* Basic email notifications. 

The code is currently in super hacky status (no make file or build script), and is specific to my website in some key places. Pulling the hostname into the config file is the next item on the list. (grep for junglecoder to get an idea of which files will be affected.)

Choices currently made for this engine:

* Hand rolled, Akismet filtered comments 
* Simple theme 
* Low amount of dynamisim
* Powered by a Sqlite file 
* Log to stderr, redirect when launching from cmd line

##Current state of deployment (to be made easier)
To set it up on your VPS/server you will need to at least do the following:

1- Make sure that you have Sqlite3 (the `libsqlite3-dev` package on apt) and [golang](http://golang.org/doc/install) installed. 

2- Run `go get` for the following dependencies 

	code.google.com/p/go.crypto/bcrypt
 	github.com/gorilla/feeds
 	github.com/russross/blackfriday
 	github.com/tgascoigne/akismet
 	github.com/yumaikas/blogserv/config
 	github.com/yumaikas/blogserv/WebAdmin
 	github.com/yumaikas/golang-die

3- Create a config file with the following form:

```json 
{
  "AkismetKey": "KeyHere",
  "WebRoot": "$GOPATH/github.com/yumaikas/blogserv/webroot/",
  "DBPath": "yourDBfileLocation",
  "TemplatePath": "$GOPATH/github.com/yumaikas/blogserv/Templates",
  "PostPath": "",
  "NotificationConfig" : {
	"EmailAddress": "",
	"ToBeNotified": ["tobenotifed@gmail.com"],
	"PlainAuth" : {
		"Identity": "",
		"Username": "example@gmail.com",
		"Password": "passw0rd",
		"Host"    : "smtp.gmail.com:587"
	}
  }
}
```

4- Create an environment variable named `BLOGSERV_CONFIG` that points to the json file created above

5- Create the Sqlite Databse file with the following schema(will be updated as the blog changes): and point to the Sqlite file in the config file.

```Sql

	CREATE TABLE Articles (URL TEXT, id INTEGER PRIMARY KEY, Content TEXT, Title TEXT, PublishStage TEXT);
	CREATE TABLE Comments (id INTEGER PRIMARY KEY, ArticleID int, UserID int, Content string, GUID TEXT, Status TEXT);
	CREATE TABLE Tags (id INTEGER PRIMARY KEY, name TEXT, ArticleID NUMERIC);
	CREATE TABLE Users (Email TEXT, id INTEGER PRIMARY KEY, screenName TEXT);
	CREATE TABLE Visits (IPAddress TEXT, UserID NUMERIC, UserAgent TEXT);
	CREATE TABLE authToken (token TEXT, userID TEXT, IPAddress TEXT);
	CREATE TABLE credentials(password TEXT, userID TEXT, actions TEXT
	);
```

6- Run `go build` and `go install` in `blogserv/admin`, `blogserv/postMerger`, and `blogserv`

7- Create the site's first article in the following form (sample title), saving it as an `*.mdown` file in the directory that you put in the config for blog articles:

	"URL":"AboutMe"
	"Title":"About Me"
	Text:{
		The blog works!
		}:Text

8- Run `postMerger` to add the article to the website.

9- Run `nohup blogserv &> log` to test the server on linux, or just run `blogserv.exe` if on Windows. 

10-  If all that worked, the blog should be listeing on port 6060.

## Extended setup

To be able to create and edit articles as a blog administrator, you will need to setup a web account on the server. If the above has completed succesfully, then all that needs to be done is to use the admin command(use a space in front of the command to keep it from going to the bash history):

```bash
 admin -createUser -password=passwordhere -userid
```
