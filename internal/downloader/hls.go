package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	neturl "net/url"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"deps.me/chzzk-vod-dl/internal/api"
	"golang.org/x/sync/semaphore"
)

type StatusWriter struct {
	TotalSegmentLen int64
	SegmentCounter  atomic.Int64
}

func (w *StatusWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\r")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			count := w.SegmentCounter.Load()
			total := w.TotalSegmentLen

			percent := int64(0)
			if total > 0 {
				percent = (count * 100) / total
			}
			fmt.Printf("\r%s | segments=%d/%d (%d%%)\033[K", line, count, total, percent)
		}
	}
	return len(p), nil
}

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

	statusWriter := StatusWriter{
		TotalSegmentLen: int64(len(playlist.Segments)),
		SegmentCounter:  atomic.Int64{},
	}

	cmd := exec.Command("ffmpeg",
		"-hide_banner",
		"-i", "pipe:0",
		"-c", "copy",
		"-movflags", "+frag_keyframe+empty_moov+default_base_moof+global_sidx",
		"-y",
		outputName,
	)

	cmd.Stdout = &statusWriter
	cmd.Stderr = &statusWriter

	ffmpegStdin, err := cmd.StdinPipe()
	if err != nil {
		slog.Error("DownloadHLSVideo cmd.StdinPipe", "error", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		slog.Error("DownloadHLSVideo cmd.Start", "error", err)
		return err
	}

	maxWorkers := int64(runtime.GOMAXPROCS(0) * 2)
	sem := semaphore.NewWeighted(maxWorkers)

	type segmentResult struct {
		index int
		data  []byte
		err   error
	}
	resultChan := make(chan segmentResult, maxWorkers)

	doneChan := make(chan error, 1)
	go func() {
		nextExpected := -1
		buffer := make(map[int][]byte)

		for {
			res := <-resultChan
			if res.err != nil {
				doneChan <- res.err
				return
			}
			buffer[res.index] = res.data

			for {
				data, ok := buffer[nextExpected]
				if !ok {
					break
				}

				_, err := ffmpegStdin.Write(data)
				if err != nil {
					doneChan <- fmt.Errorf("ffmpeg write error: %s", err)
					return
				}

				delete(buffer, nextExpected)
				nextExpected++

				if nextExpected == len(playlist.Segments) {
					ffmpegStdin.Close()
					doneChan <- nil
					return
				}
			}
		}
	}()

	go func() {
		res, err := client.GetWithRetry(playlist.Init, 10)
		if err != nil {
			resultChan <- segmentResult{err: err}
			return
		}
		data, _ := io.ReadAll(res.Body)
		res.Body.Close()
		resultChan <- segmentResult{index: -1, data: data}
	}()

	for i, url := range playlist.Segments {
		sem.Acquire(context.Background(), 1)
		go func(i int, url string) {
			defer sem.Release(1)

			res, err := client.GetWithRetry(url, 10)
			if err != nil {
				resultChan <- segmentResult{err: err}
				return
			}

			data, _ := io.ReadAll(res.Body)
			res.Body.Close()
			resultChan <- segmentResult{index: i, data: data}
			statusWriter.SegmentCounter.Add(1)
		}(i, url)
	}

	if err := <-doneChan; err != nil {
		cmd.Process.Kill()
		return err
	}

	return cmd.Wait()
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

