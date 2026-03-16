package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"deps.me/chzzk-vod-dl/internal/api"
	"deps.me/chzzk-vod-dl/internal/db"
	"deps.me/chzzk-vod-dl/internal/downloader"
	"deps.me/chzzk-vod-dl/internal/utils"
)

var commitHash string

func main() {
	fmt.Printf("cvd (chzzk-vod-dl) by hajin %s\n", commitHash)
	if len(os.Args) < 2 {
		PrintHelp()
		return
	}

	db, err := db.OpenDatabase()
	if err != nil {
		slog.Error("main OpenDatabase", "error", err)
	}

	aut, err := db.GetValue("NID_AUT")
	if err != nil {
		aut = ""
	}
	ses, err := db.GetValue("NID_SES")
	if err != nil {
		ses = ""
	}

	client := api.NewChzzkClient(aut, ses)

	slog.SetLogLoggerLevel(slog.LevelInfo)

	cmd := os.Args[1]
	switch cmd {
	case "list":
		HandleList(client)
	case "info":
		HandleInfo(client)
	case "download":
		HandleDownload(client, db)
	case "all":
		HandleAll(client, db)
	case "auth":
		HandleAuth(client, db)
	case "me":
		HandleMe(client)
	case "archive":
		HandleArchive(db)
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
	fmt.Println("  cvd auth <NID_AUT> <NID_SES>")
	fmt.Println("  cvd me")
	fmt.Println("  cvd archive list")
	fmt.Println("  cvd archive remove <video no>")
}

func HandleList(client *api.ChzzkClient) {
	// parse arguments
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	channelId := os.Args[2]

	videos, err := client.GetVideoList(channelId)
	if err != nil {
		slog.Error("HandleList GetVideoList", "error", err)
		return
	}

	fmt.Printf("Video list [%s]\n", channelId)
	for _, video := range videos {
		date, err := utils.FormatDate(video.Date)
		if err != nil {
			slog.Error("HandleList FormatDate", "error", err)
			return
		}
		duration := utils.FormatDuration(video.Duration)
		fmt.Printf("%d\t%s\t%s\t%s\n", video.VideoNo, date, duration, video.Title)
	}
}

func HandleInfo(client *api.ChzzkClient) {
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

	info, err := client.GetVideoInfo(videoNo)
	if err != nil {
		slog.Error("HandleInfo GetVideoInfo", "error", err)
		return
	}

	date, err := utils.FormatDate(info.Date)
	if err != nil {
		slog.Error("HandleInfo FormatDate", "error", err)
		return
	}
	fmt.Printf("%-8d %10s %s\n", info.VideoNo, date, info.Title)
}

func HandleDownload(client *api.ChzzkClient, db *db.DB) {
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	videoNo, err := strconv.Atoi(os.Args[2])
	if err != nil {
		slog.Error("HandleDownload strconv.Atoi", "error", err)
		return
	}

	info, err := downloader.DownloadVideo(client, videoNo)
	if err != nil {
		slog.Error("HandleDownload DownloadVideo", "error", err)
		return
	}
	if err = db.RecordDownload(videoNo, info.Title); err != nil {
		slog.Error("HandleDowlnoad db.RecordDownload", "error", err)
		return
	}
}

func HandleAll(client *api.ChzzkClient, db *db.DB) {
	if len(os.Args) < 3 {
		fmt.Println("Failed to parse list command")
		os.Exit(1)
	}
	channelId := os.Args[2]

	videos, err := client.GetVideoList(channelId)
	if err != nil {
		slog.Error("HandleAll GetVideoList", "error", err)
		return
	}

	for _, video := range videos {
		chk, err := db.CheckDownload(video.VideoNo)
		if  err == nil && chk == false {
			info, err := downloader.DownloadVideo(client, video.VideoNo)
			if err != nil {
				slog.Error("HandleDownload DownloadVideo", "error", err)
				return
			}
			if err = db.RecordDownload(video.VideoNo, info.Title); err != nil {
				slog.Error("HandleDowlnoad db.RecordDownload", "error", err)
				return
			}
		}
	}
}

func HandleAuth(client *api.ChzzkClient, db *db.DB) {
	if len(os.Args) != 4 && len(os.Args) != 2 {
		fmt.Println("failed to parse auth command")
		os.Exit(1)
	}

	if len(os.Args) == 4 {
		aut := os.Args[2]
		err := db.SetValue("NID_AUT", aut)
		if err != nil {
			slog.Error("HandleAuth db.SetValue", "error", err)
			return
		}

		ses := os.Args[3]
		err = db.SetValue("NID_SES", ses)
		if err != nil {
			slog.Error("HandleAuth db.SetValue", "error", err)
			return
		}

		fmt.Printf("Successfully saved\n")

		client.Auth(aut, ses)
	}

	HandleMe(client)
}

func HandleMe(client *api.ChzzkClient) {
	userStatus, err := client.GetUserStatus()
	if err != nil {
		slog.Error("HandleMe client.GetUserStatus", "err", err)
		return
	}

	if userStatus.Content.NickName != nil {
		fmt.Printf("Authorized as %s\n", *userStatus.Content.NickName)
	} else {
		fmt.Printf("Failed to get user status\n")
	}
}

func HandleArchive(db *db.DB) {
	if len(os.Args) < 3 {
		PrintHelp()
		return
	}

	cmd := os.Args[2]
	switch cmd {
	case "remove", "rm":
		HandleArchiveRemove(db)
	case "list", "ls":
		HandleArchiveList(db)
	default:
		PrintHelp()
	}
}

func HandleArchiveList(db *db.DB) {
	downloads := db.ListDownload()
	for _, download := range downloads {
		fmt.Printf("%s\t%s\n", download.VideoId, download.Title)
	}
}

func HandleArchiveRemove(db *db.DB) {
	if len(os.Args) < 4 {
		fmt.Printf("Failed to parse archive remove command")
	}

	videoId := os.Args[3]
	err := db.RemoveDownload(videoId)
	if err != nil {
		slog.Error("HandleArchiveRemove db.RemoveDownload", "error", err)
		return
	}
	fmt.Printf("Successfully removed %s\n", videoId)
}

