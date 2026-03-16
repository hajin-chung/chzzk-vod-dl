package downloader

import (
	"bufio"
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

	"deps.me/chzzk-vod-dl/internal/api"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/semaphore"
)

func DownloadHLSVideo(client *api.ChzzkClient, videoUrl string, outputName string) error {
	// parse hls
	playlistUrl, err := getPlaylistUrl(client, videoUrl)
	if err != nil {
		slog.Error("DownloadHLSVideo parseMasterHLS", "error", err)
		return err
	}

	playlistHLS, err := client.GetBody(playlistUrl)
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

	if err := downloadFileRetry(client, playlist.Init, ".tmp/init.m4s", 10, -1); err != nil {
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

			if err := downloadFileRetry(client, url, fmt.Sprintf(".tmp/%d.m4v", i), 10, i); err != nil {
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
	target, err := os.OpenFile(".tmp/all.mp4", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer target.Close()

	// Wrap in a buffer (e.g., 1MB buffer)
	writer := bufio.NewWriterSize(target, 1024*1024)

	// Helper function to append files
	appendFile := func(path string) error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		// Closures ensure the file is closed immediately after the copy
		defer f.Close() 

		_, err = io.Copy(writer, f)
		return err
	}

	// 2. Copy Init Segment
	if err := appendFile(".tmp/init.m4s"); err != nil {
		slog.Error("Failed to copy init segment", "error", err)
		return err
	}

	// 3. Copy Segments
	bar = progressbar.Default(int64(len(playlist.Segments)), "concat")
	for i := range playlist.Segments {
		path := fmt.Sprintf(".tmp/%d.m4v", i)
		if err := appendFile(path); err != nil {
			slog.Error("Failed to copy segment", "index", i, "error", err)
			return err
		}
		bar.Add(1)
	}

	// 4. IMPORTANT: Flush the buffer to disk before closing the underlying file
	writer.Flush()

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

func getPlaylistUrl(client *api.ChzzkClient, url string) (string, error) {
	playlistHLS, err := client.GetBody(url)
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

func downloadFile(client *api.ChzzkClient, url string, path string) error {
	res, err := client.Get(url)
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

func downloadFileRetry(client *api.ChzzkClient, url string, path string, retry int, index int) error {
	var err error
	for retry > 0 {
		err = downloadFile(client, url, path)
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

