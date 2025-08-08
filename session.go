package main

import (
	"os"
	"strings"
)

var session string = ""
var sessionLoadSuccess bool = false

func LoadSession() error {
	bytes, err := os.ReadFile(sessionPath)
	if err != nil {
		return err
	}

	session = strings.Trim(string(bytes[:]), "\n")
	sessionLoadSuccess = true
	return nil
}
