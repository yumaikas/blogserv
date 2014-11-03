// This pacakge get's its values froma config file.

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"sync"
)

var (
	ErrConfigNotFound    error = errors.New("Config file not found at path from Environment variable. Check the value of $BLOGSERV_CONFIG")
	ErrInvalidEnvPath    error = errors.New("The environment variable was not found")
	ErrInvalidConfigFile error = errors.New("The config file was in the wrong format")
)

type blogservConfig struct {
	AkismetKey        string             "AkismetKey"
	WebRoot           string             "WebRoot"
	DbPath            string             "DbPath"
	TemplatePath      string             "TemplatePath"
	PostPath          string             "PostPath"
	EmailNotifyConfig NotificationConfig `json:"NotificationConfig"`
}

type NotificationConfig struct {
	EmailAddress string           "EmailAddress"
	ToBeNotified []string         "ToBeNotified"
	PlainAuth    NotificaitonAuth "PlainAuth"
}

type NotificaitonAuth struct {
	Identity string "Identity"
	UserName string "Username"
	Password string "Password"
	Host     string "Host"
}

func init() {
	var err error
	conf, err = defaultConfig()
	if err != nil {
		panic("Config not found! :" + err.Error())
	}
}

var conf *blogservConfig
var m = new(sync.RWMutex)

func AkismetKey() string {
	m.RLock()
	defer m.RUnlock()
	return conf.AkismetKey
}
func WebRoot() string {
	m.RLock()
	defer m.RUnlock()
	return conf.WebRoot
}
func DbPath() string {
	m.RLock()
	defer m.RUnlock()
	return conf.DbPath
}
func TemplatePath() string {
	m.RLock()
	defer m.RUnlock()
	return conf.TemplatePath
}
func PostPath() string {
	m.RLock()
	defer m.RUnlock()
	return conf.PostPath
}

type EmailInfo struct {
	Auth         smtp.Auth
	ToBeNotified []string
	HostServer   string // Needs to be a valid server.com:portNum combo
	FromEmail    string
}

func EmailAuth() EmailInfo {
	m.RLock()
	defer m.RUnlock()
	p := conf.EmailNotifyConfig.PlainAuth

	// slice to the last instace of :, so as to get the address
	host := p.Host[:strings.LastIndex(p.Host, ":")]
	eConf := EmailInfo{
		smtp.PlainAuth(p.Identity, p.UserName, p.Password, host),
		conf.EmailNotifyConfig.ToBeNotified,
		p.Host,
		conf.EmailNotifyConfig.EmailAddress,
	}
	return eConf
}

func configPath() string {
	return os.Getenv("BLOGSERV_CONFIG")
}

func ReloadConfig() {
	config, err := defaultConfig()
	if err != nil {
		return
	}
	m.Lock()
	conf = config
	m.Unlock()
}

const localPath = `config.json`

// Look in the current folder for a config file, and then in location specified by the environment variable
func defaultConfig() (*blogservConfig, error) {
	// Return default config based on settings
	var p string
	os.Stat(localPath)
	if info, err := os.Stat(localPath); err == nil && info.Mode().IsRegular() {
		p = localPath
	} else if p = os.Getenv("BLOGSERV_CONFIG"); p == "" {
		return nil, ErrInvalidEnvPath
	}
	var f *os.File
	defer f.Close()
	var err error
	if f, err = os.Open(p); err != nil {
		return nil, ErrConfigNotFound
	}
	cf := json.NewDecoder(f)
	var config blogservConfig
	if err := cf.Decode(&config); err != nil {
		fmt.Println(err.Error())
		return nil, ErrInvalidConfigFile
	}
	return &config, nil
}
