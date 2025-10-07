package main

import (
	"log/slog"
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

	userStatus, err := GetUserStatus()
	if err != nil {
		slog.Error("LoadSession GetUserStatus", "error", err)
	} else if !userStatus.Content.HasProfile {
		slog.Error("LoadSession", "error", "session is old update!!!")
	}
	slog.Info("LoadSession", "user status", userStatus)
	return nil
}

