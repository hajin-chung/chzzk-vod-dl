package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
)

func main() {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)
	downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)

	if len(os.Args) < 2 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "list":
		handleList(listCmd)
	case "info":
		handleInfo(infoCmd)
	case "download":
		handleDownload(downloadCmd)
	case "new":
		handleNew(newCmd)
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

func handleList(listCmd *flag.FlagSet) {
	if err := listCmd.Parse(os.Args[2:]); err != nil {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	if listCmd.NArg() != 1 {
		fmt.Println("You must provide a channel id")
		os.Exit(1)
	}
	channelId := listCmd.Arg(0)
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

func handleInfo(infoCmd *flag.FlagSet) {
	if err := infoCmd.Parse(os.Args[2:]); err != nil {
		fmt.Println("Failed to parse info command")
		os.Exit(1)
	}
	if infoCmd.NArg() != 1 {
		fmt.Println("You must provide a video number")
		os.Exit(1)
	}

	videoNo, err := strconv.Atoi(infoCmd.Arg(0))
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

func handleDownload(downloadCmd *flag.FlagSet) {
	if err := downloadCmd.Parse(os.Args[2:]); err != nil {
		fmt.Println("Failed to parse download command")
		os.Exit(1)
	}
	if downloadCmd.NArg() != 1 {
		fmt.Println("You must provide a video number")
		os.Exit(1)
	}

	videoNo, err := strconv.Atoi(downloadCmd.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	DownloadVideo(videoNo)
}

func handleNew(cmd *flag.FlagSet) {
	if err := cmd.Parse(os.Args[2:]); err != nil {
		fmt.Println("Failed to parse new command")
		os.Exit(1)
	}
	if cmd.NArg() != 1 {
		fmt.Println("You must provide a channelId")
		os.Exit(1)
	}

	channelId := cmd.Arg(0)
	fmt.Printf("Check new video on channel %s\n", channelId)

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

func DownloadVideo(videoNo int) error {
	fmt.Printf("Download [%d]\n", videoNo)

	info, err := getVideoInfo(videoNo)
	if err != nil {
		return err
	}
	date, err := formatDate(info.Date)
	if err != nil {
		return err
	}
	outputName := fmt.Sprintf("%s %s.mp4", date, info.Title)

	dashUrl, err := getDashUrl(videoNo)
	if err != nil {
		return err
	}

	fmt.Printf("dash url: %s\n", dashUrl)

	videoUrl, err := getVideoUrl(dashUrl)
	if err != nil {
		return err
	}

	fmt.Printf("video url: %s\n", videoUrl)

	cmd := exec.Command("axel", "-n", "8", "-o", outputName, videoUrl)
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
