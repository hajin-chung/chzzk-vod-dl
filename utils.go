package main

import (
	"strings"
	"time"
)

func formatDate(rawDate string) (string, error) {
	date, err := time.Parse("2006-01-02 15:04:05", rawDate)
	if err != nil {
		return "", err
	}

	return date.Format("06.01.02"), nil
}

func sanitizeFileName(name string) string {
	sanitized := name
	for _, char := range strings.Split(`\/:*?"<>|`, "") {
		sanitized = strings.ReplaceAll(sanitized, char, "")
	}
	return sanitized
}
