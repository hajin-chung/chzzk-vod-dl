package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}
	
	err := LoadSession()
	if err != nil {
		log.Printf("error while loading session: %s\n", err)
	}

	cmd := os.Args[1]
	switch cmd {
	case "list":
		handleList()
	case "info":
		handleInfo()
	case "download":
		handleDownload()
	case "new":
		handleNew()
	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("cvd [Chzzk VOD Downloader]")
	fmt.Println("Usage:")
	fmt.Println("  cvd list <channel id>")
	fmt.Println("  cvd info <video no>")
	fmt.Println("  cvd download <video no>")
	fmt.Println("  cvd new <channel id>")
}

func handleList() {
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	channelId := os.Args[2]
	fmt.Printf("Video list [%s]\n", channelId)

	videos, err := getVideoList(channelId)
	if err != nil {
		log.Fatal(err)
	}

	for _, video := range videos {
		date, err := formatDate(video.Date)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-8d %10s %s\n", video.VideoNo, date, video.Title)
	}
}

func handleInfo() {
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	videoNo, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Info [%d]\n", videoNo)

	info, err := getVideoInfo(videoNo)
	if err != nil {
		log.Fatal(err)
	}

	date, err := formatDate(info.Date)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%-8d %10s %s\n", info.VideoNo, date, info.Title)
}

func handleDownload() {
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	videoNo, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

	err = DownloadVideo(videoNo)
	if err != nil {
		log.Fatal(err)
	}
}

func handleNew() {
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	channelId := os.Args[2]

	videos, err := getVideoList(channelId)
	if err != nil {
		log.Panic(err)
	}

	newVideoFound := false
	for _, video := range videos {
		if CheckMemo(video.VideoNo) == false {
			newVideoFound = true
			DownloadVideo(video.VideoNo)
			break
		}
	}

	if newVideoFound == false {
		fmt.Printf("new video not found")
		os.Exit(4)
	}
}

func DownloadVideo(videoNo int, args ...string) error {
	info, err := getVideoInfo(videoNo)
	if err != nil {
		return err
	}
	date, err := formatDate(info.Date)
	if err != nil {
		return err
	}
	outputName := sanitizeFileName(fmt.Sprintf("/vod/%s %s.mp4", date, info.Title))

	dashUrl, err := getDashUrl(videoNo)
	if err != nil {
		return err
	}

	videoUrl, err := getVideoUrl(dashUrl)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n%s\n", videoUrl, outputName)

	command := []string{"-n", "8", "-o", outputName, videoUrl}
	command = append(command, args...)
	cmd := exec.Command("axel", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	if err := AddMemo(videoNo); err != nil {
		return err
	}
	return nil
}
