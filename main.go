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
		PrintHelp()
		return
	}
	
	LoadEnv()
	err := LoadSession()
	if err != nil {
		log.Printf("error while loading session: %s\n", err)
	}

	cmd := os.Args[1]
	switch cmd {
	case "list":
		HandleList()
	case "info":
		HandleInfo()
	case "download":
		HandleDownload()
	case "new":
		HandleNew()
	default:
		PrintHelp()
	}
}

func PrintHelp() {
	fmt.Println("cvd [Chzzk VOD Downloader]")
	fmt.Println("Usage:")
	fmt.Println("  cvd list <channel id>")
	fmt.Println("  cvd info <video no>")
	fmt.Println("  cvd download <video no>")
	fmt.Println("  cvd new <channel id>")
}

func HandleList() {
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	channelId := os.Args[2]
	fmt.Printf("Video list [%s]\n", channelId)

	videos, err := GetVideoList(channelId)
	if err != nil {
		log.Fatal(err)
	}

	for _, video := range videos {
		date, err := FormatDate(video.Date)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-8d %10s %s\n", video.VideoNo, date, video.Title)
	}
}

func HandleInfo() {
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	videoNo, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Info [%d]\n", videoNo)

	info, err := GetVideoInfo(videoNo)
	if err != nil {
		log.Fatal(err)
	}

	date, err := FormatDate(info.Date)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%-8d %10s %s\n", info.VideoNo, date, info.Title)
}

func HandleDownload() {
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

func HandleNew() {
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	channelId := os.Args[2]

	videos, err := GetVideoList(channelId)
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
	info, err := GetVideoInfo(videoNo)
	if err != nil {
		return err
	}
	date, err := FormatDate(info.Date)
	if err != nil {
		return err
	}
	outputName := "vod/"+SanitizeFileName(fmt.Sprintf("%s %s.mp4", date, info.Title))

	dashUrl, err := GetDashUrl(videoNo)
	if err != nil {
		return err
	}

	videoUrl, err := GetVideoUrl(dashUrl)
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
