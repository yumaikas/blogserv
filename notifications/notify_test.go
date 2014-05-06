package notifications

import (
	"testing"

	arts "github.com/yumaikas/blogserv/blogArticles"
)

func TestSend(t *testing.T) {
	sendEmail([]comment{
		comment{
			arts.Comment{},
			"TestingEmail",
			"TestingIPAddr",
			"TestingURL",
			"TestingName",
		},
		comment{
			arts.Comment{},
			"test@gmail.com",
			"::1",
			"AboutMe",
			"About Me",
		},
	})
}
