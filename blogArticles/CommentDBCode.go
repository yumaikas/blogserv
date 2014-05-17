package blogArticles

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tgascoigne/akismet"
	"github.com/yumaikas/blogserv/WebAdmin"
)

var duplicateGUIDError = errors.New("Comment GUID already exists in database")

type Comment struct {
	UserName, Content, Status, GUID string
}

func execGUIDquery(query, guid string) error {
	db, err := dbOpen()
	defer db.Close()
	if err != nil {
		return err
	}
	_, err = db.Exec(query, guid)
	return err
}
func ShowComment(guid string) error {
	return execGUIDquery(`Update Comments Set Status = 'Shown' where guid = ?`, guid)
}

func DeleteComment(guid string) error {
	return execGUIDquery(`Update Comments Set Status = 'Deleted' where guid = ?`, guid)
}

func HideComment(guid string) error {
	return execGUIDquery(`Update Comments Set Status = 'Hidden' where guid = ?`, guid)
}

func CommentToDB(c akismet.Comment, arName string) error {
	fmt.Print("Enter CommentToDB")
	defer fmt.Print("Exit CommentToDB")
	db, err := dbOpen()
	defer db.Close()
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	rb := func(e error) error {
		tx.Rollback()
		return e
	}
	if err != nil {
		return rb(err)
	}

	//This is what is going in to the db
	in := struct {
		UserID, ArticleID int
		Content           string
	}{0, 0, c.Content}

	err = tx.QueryRow(`Select id from Users where Email = ?`, c.AuthorEmail).Scan(&in.UserID)
	switch {
	case err == sql.ErrNoRows:
		var u_err error
		in.UserID, u_err = addUser(c, tx)
		if u_err != nil {
			return rb(err)
		}
		break
	case err != nil:
		return rb(err)
	}
	err = tx.QueryRow(`Select id from Articles where URL = ?`, arName).Scan(&in.ArticleID)
	if err != nil {
		return rb(err)
	}
	guid, err := NewCommentGuid(tx)
	if err != nil {
		return rb(err)
	}
	//The results and error(if any)
	q := `Insert into Comments (UserID, ArticleID, Content, GUID, Status) 
			values (?, ?, ?, ?, ?)`
	r, err := tx.Exec(q, in.UserID, in.ArticleID, in.Content, guid, "Shown")
	if err != nil {
		return rb(err)
	}
	numRows, err := r.RowsAffected()
	switch {
	case err != nil:
		return rb(err)
	case numRows != 1:
		return rb(fmt.Errorf("Error: %d rows were affected instead of 1", numRows))
	}
	err = tx.Commit()
	if err != nil {
		return rb(err)
	}
	return nil
}

func generateCommentGuid(db *sql.Tx) (string, error) {
	commentGUID, err := WebAdmin.GenerateRandomString()
	if err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}
	guidCount := 0
	q := `Select Count(id) from Comments where GUID = ?`
	err = db.QueryRow(q, commentGUID).Scan(&guidCount)
	if err != nil {
		return "", err
	} else if guidCount > 0 {
		return "", duplicateGUIDError
	}
	return commentGUID, nil
}

func NewCommentGuid(tx *sql.Tx) (string, error) {
	for {
		guid, err := generateCommentGuid(tx)
		//only try again if a duplicate was created
		//Pass other error up, or return successfully
		if err == duplicateGUIDError {
			continue
		}
		return guid, err
	}
}
