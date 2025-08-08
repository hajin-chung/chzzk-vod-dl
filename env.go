package main

import (
	"os"
	"log"
)

var sessionPath string = "etc/session.txt"
var memoPath string = "etc/memo.txt"

func LoadEnv() {
	osSessionPath := os.Getenv("CVDL_SESSION")
	if len(osSessionPath) > 0 {
		sessionPath = osSessionPath
	}
	osMemoPath := os.Getenv("CVDL_MEMO")
	if len(osMemoPath) > 0 {
		memoPath = osMemoPath
	}

	log.Println("Loaded environment variables")
	log.Printf("CVDL_SESSION=%s\n", sessionPath)
	log.Printf("CVDL_MEMO=%s\n", memoPath)
}
