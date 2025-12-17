package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	neturl "net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/semaphore"
)

func DownloadHLSVideo(videoUrl string, outputName string) error {
	// parse hls
	playlistUrl, err := getPlaylistUrl(videoUrl)
	if err != nil {
		slog.Error("DownloadHLSVideo parseMasterHLS", "error", err)
		return err
	}

	playlistHLS, err := GetBody(playlistUrl)
	if err != nil {
		slog.Error("DownloadHLSVideo GetBody", "error", err)
		return err
	}

	playlist, err := parsePlaylistHLS(playlistUrl, playlistHLS)
	if err != nil {
		slog.Error("DownloadHLSVideo parsePlaylistHLS", "error", err)
		return err
	}

	os.RemoveAll(".tmp")
	// download files
	if err := os.MkdirAll(".tmp", 0755); err != nil {
		slog.Error("DownloadHLSVideo os.MkdirAll", "error", err)
		return err
	}
	defer os.RemoveAll(".tmp")

	if err := downloadFileRetry(playlist.Init, ".tmp/init.m4s", 10, -1); err != nil {
		slog.Error("DownloadHLSVideo downloadFile", "error", err)
		return err
	}

	maxWorkers := int64(runtime.GOMAXPROCS(0))
	sem := semaphore.NewWeighted(maxWorkers)

	slog.Debug("DownloadHLSVideo", "segment length", len(playlist.Segments))
	bar := progressbar.Default(int64(len(playlist.Segments)), "download")
	for i, url := range playlist.Segments {
		if err := sem.Acquire(context.Background(), 1); err != nil {
			slog.Error("downloadFile sem.Acquire", "error", err)
		}
		go func(i int, url string, sem *semaphore.Weighted) {
			defer sem.Release(1)

			if err := downloadFileRetry(url, fmt.Sprintf(".tmp/%d.m4v", i), 10, i); err != nil {
				slog.Error("DownloadHLSVideo downloadFile", "error", err)
				fmt.Printf("download segment %d / %d FAIL\n", i, len(playlist.Segments))
			}
			bar.Add(1)
		}(i, url, sem)
	}

	if err := sem.Acquire(context.Background(), maxWorkers); err != nil {
		slog.Error("Download", "error", err)
		return err
	}

	// concat segments
	file, err := os.OpenFile(".tmp/all.mp4", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		slog.Error("DownloadHLSVideo os.OpenFile", "error", err)
		return err
	}
	defer file.Close()

	f, err := os.Open(".tmp/init.m4s")
	if err != nil {
		slog.Error("DownloadHLSVideo os.Open", "error", err)
		return err
	}
	defer f.Close()

	_, err = io.Copy(file, f)
	if err != nil {
		slog.Error("DownloadHLSVideo io.Copy", "error", err)
		return err
	}

	bar = progressbar.Default(int64(len(playlist.Segments)), "concat")
	for i := range len(playlist.Segments) {
		sf, err := os.Open(fmt.Sprintf(".tmp/%d.m4v", i))
		if err != nil {
			slog.Error("DownloadHLSVideo os.Open", "error", err)
			return err
		}

		_, err = io.Copy(file, sf)
		if err != nil {
			slog.Error("DownloadHLSVideo io.Copy", "error", err)
			return err
		}
		bar.Add(1)
	}

	// remux
	cmd := exec.Command("ffmpeg", "-i", ".tmp/all.mp4", "-c", "copy", "-y", outputName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		slog.Error("DownloadHLSVideo cmd.Run", "error", err)
		return err
	}

	return nil
}

func getPlaylistUrl(url string) (string, error) {
	playlistHLS, err := GetBody(url)
	if err != nil {
		slog.Error("getPlaylistUrl GetBody", "error", err)
		return "", err
	}

	bandwidthRe := regexp.MustCompile(`BANDWIDTH=(\d+),`)
	lines := strings.Split(playlistHLS, "\n")
	maxBandwidth := 0
	playlistPath := ""
	for i := 0; i < len(lines); i++ {
		match := bandwidthRe.FindStringSubmatch(lines[i])
		if len(match) < 2 {
			continue
		}

		bandwidth, err := strconv.Atoi(match[1])
		if err != nil {
			slog.Error("getPlaylistUrl strconv.Atoi", "error", "err", "match", match[1])
		}

		if bandwidth > maxBandwidth {
			maxBandwidth = bandwidth
			i += 1
			playlistPath = lines[i]
		}
	}

	if playlistPath == "" {
		return "", errors.New("no playlist url found")
	}
	slog.Info("getPlaylistUrl", "playlistPath", playlistPath)

	playlistUrl, err := UrlJoin(url, playlistPath)
	if err != nil {
		slog.Error("getPlaylistUrl neturl.JoinPath", "error", err)
		return "", err
	}

	return playlistUrl, nil
}

type playlist struct {
	Init     string
	Segments []string
}

func parsePlaylistHLS(url string, hls string) (*playlist, error) {
	initRe := regexp.MustCompile(`#EXT-X-MAP:URI="(.+)"`)
	match := initRe.FindStringSubmatch(hls)
	if len(match) < 2 {
		return nil, errors.New("no init uri found")
	}
	slog.Info("t", "match[1]", match[1])
	initUrl, err := UrlJoin(url, match[1])
	if err != nil {
		return nil, err
	}

	segments := []string{}
	segmentRe := regexp.MustCompile(`#EXTINF:`)
	lines := strings.Split(hls, "\n")
	for i := 0; i < len(lines); i++ {
		match = segmentRe.FindStringSubmatch(lines[i])
		if len(match) < 1 {
			continue
		}
		url, err := UrlJoin(url, lines[i+1])
		if err != nil {
			return nil, err
		}
		segments = append(segments, url)
		i += 1
	}

	return &playlist{initUrl, segments}, nil
}

func downloadFile(url string, path string) error {
	res, err := Get(url)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	bytesWritten, err := io.Copy(file, res.Body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	slog.Debug("downloadFile", "path", path, "bytesWritten", bytesWritten)
	return nil
}

func downloadFileRetry(url string, path string, retry int, index int) error {
	var err error
	for retry > 0 {
		err = downloadFile(url, path)
		if err == nil {
			return nil
		}
		slog.Error("downloadFileRetry downloadFile error retry", "left", retry, "index", index)
		retry--
	}
	return err
}

func UrlJoin(base string, part string) (string, error) {
	url, err := neturl.Parse(base)
	if err != nil {
		return "", err
	}
	partUrl, err := neturl.Parse(part)
	if err != nil {
		return "", err
	}
	url = url.ResolveReference(partUrl)

	return url.String(), nil
}

