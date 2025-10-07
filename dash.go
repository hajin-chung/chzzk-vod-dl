package main

import (
	"os"
	"os/exec"
)

func DownloadDASHVideo(videoUrl string, outputName string) error {
	command := []string{"-n", "8", "-o", outputName, videoUrl}
	cmd := exec.Command("axel", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}
	
	return nil
}
