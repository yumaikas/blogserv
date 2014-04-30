package config

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestConfig(t *testing.T) {
	if c, err := defaultConfig(); c == nil || err != nil {
		t.Log(configPath())
		t.Log(err)
		t.Fail()
	} else {
		t.Log(c)
	}
}

func TestForFile(t *testing.T) {
	buf, _ := ioutil.ReadFile(configPath())
	if len(buf) == 0 {
		t.Log("Empty config file")
		t.Fail()
	}
	t.Log(string(buf))
}

func TestJsonMarshal(t *testing.T) {
	b := blogservConfig{
		"akismetKey",
		"webroot",
		"Dbpath",
		"templatePath",
		"PostPath",
		notificationConfig{
			"EmailAddress",
			[]string{"Email added"},
			NotificaitonAuth{
				"Identity",
				"UserName",
				"Password",
				"Host",
			},
		},
	}
	data, err := json.Marshal(b)
	t.Log(string(data), err)
}
