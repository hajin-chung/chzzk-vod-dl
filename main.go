package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

var commitHash string

func main() {
	fmt.Printf("cvd (chzzk-vod-dl) by hajin %s\n", commitHash)
	if len(os.Args) < 2 {
		PrintHelp()
		return
	}

	LoadEnv()
	err := LoadSession()
	if err != nil {
		slog.Error("main LoadEnv", "error", err)
	}

	cmd := os.Args[1]
	switch cmd {
	case "list":
		HandleList()
	case "info":
		HandleInfo()
	case "download":
		HandleDownload()
	case "all":
		HandleAll()
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
	fmt.Println("  cvd all <channel id>")
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
		slog.Error("HandleList GetVideoList", "error", err)
		return
	}

	for _, video := range videos {
		date, err := FormatDate(video.Date)
		if err != nil {
			slog.Error("HandleList FormatDate", "error", err)
			return
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
		slog.Error("HandleInfo strconv.Atoi", "error", err)
		return
	}
	fmt.Printf("Info [%d]\n", videoNo)

	info, err := GetVideoInfo(videoNo)
	if err != nil {
		slog.Error("HandleInfo GetVideoInfo", "error", err)
		return
	}

	date, err := FormatDate(info.Date)
	if err != nil {
		slog.Error("HandleInfo FormatDate", "error", err)
		return
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
		slog.Error("HandleDownload strconv.Atoi", "error", err)
		return
	}

	err = DownloadVideo(videoNo)
	if err != nil {
		slog.Error("HandleDownload DownloadVideo", "error", err)
		return
	}
}

func HandleAll() {
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	channelId := os.Args[2]

	videos, err := GetVideoList(channelId)
	if err != nil {
		slog.Error("HandleAll GetVideoList", "error", err)
		return
	}

	newVideoFound := false
	for _, video := range videos {
		if CheckMemo(video.VideoNo) == false {
			newVideoFound = true
			DownloadVideo(video.VideoNo)
		}
	}

	if newVideoFound == false {
		fmt.Printf("new video not found")
		os.Exit(4)
	}
}

func DownloadVideo(videoNo int) error {
	info, err := GetVideoInfo(videoNo)
	if err != nil {
		return err
	}
	date, err := FormatDate(info.Date)
	if err != nil {
		return err
	}
	outputName := SanitizeFileName(fmt.Sprintf("%s %s.mp4", date, info.Title))

	videoUrl, err := GetVideoUrl(videoNo)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] %s\n%s\n", videoUrl.Type, videoUrl.Url, outputName)

	switch videoUrl.Type {
	case HLS:
		if err := DownloadHLSVideo(videoUrl.Url, outputName); err != nil {
			return nil
		}
	case DASH:
		if err := DownloadDASHVideo(videoUrl.Url, outputName); err != nil {
			return nil
		}
	default:
		return errors.New("video type neither hls nor dash")
	}

	if err := AddMemo(videoNo); err != nil {
		return err
	}

	return nil
}

