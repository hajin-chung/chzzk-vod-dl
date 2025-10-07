package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"testing"
)

func TestHls(t *testing.T) {
	LoadEnv()
	err := LoadSession()
	if err != nil {
		t.Errorf(`LoadSession() error: %s`, err)
	}
	log.Printf("LoadSession() success")

	videoList, err := GetVideoList(TestChannelId)
	if err != nil {
		t.Errorf(`GetVideoList("%s") error: %s`, TestChannelId, err)
	}
	videoListLen := len(videoList)
	log.Printf("got video list of length: %d", videoListLen)

	videoUrl, err := GetVideoUrl(videoList[0].VideoNo)
	if err != nil {
		t.Errorf("GetVideoUrl(%d) error: %s", videoList[0].VideoNo, err)
	}
	log.Printf("videoUrl: %s", videoUrl.Url)

	playlistUrl, err := getPlaylistUrl(videoUrl.Url)
	if err != nil {
		t.Errorf("getPlaylistUrl(%s) error: %s", videoUrl, err)
	}
	log.Printf("playlist url: %s", playlistUrl)

	playlistHLS, err := GetBody(playlistUrl)
	if err != nil {
		t.Errorf("GetBody(%s) error: %s", playlistUrl, err)
	}

	log.Println(len(playlistHLS))

	playlist, err := parsePlaylistHLS(playlistUrl, playlistHLS)
	if err != nil {
		t.Errorf("parsePlaylistHLS(%s, ...) error: %s", playlistUrl, err)
	}

	log.Printf("init: %s, segment length: %d\n", playlist.Init, len(playlist.Segments))
}

func TestDownloadHLSVideo(t *testing.T) {
	LoadEnv()
	err := LoadSession()
	if err != nil {
		t.Errorf(`LoadSession() error: %s`, err)
	}
	log.Printf("LoadSession() success")

	videoList, err := GetVideoList(TestChannelId)
	if err != nil {
		t.Errorf(`GetVideoList("%s") error: %s`, TestChannelId, err)
	}
	videoListLen := len(videoList)
	log.Printf("got video list of length: %d", videoListLen)

	videoNo := videoList[len(videoList)-2].VideoNo
	videoUrl, err := GetVideoUrl(videoNo)
	if err != nil {
		t.Errorf("GetVideoUrl(%d) error: %s", videoList[0].VideoNo, err)
	}
	log.Printf("videoUrl: %s", videoUrl.Url)
	log.Printf("videoNo: %d", videoNo)

	err = DownloadHLSVideo(videoUrl.Url, "test.mp4")
	if err != nil {
		t.Errorf("DownloadHLSVideo(%s, ...) error: %s", videoUrl.Url, err)
	}
}

func TestConcat(t *testing.T) {
	n := 4671

	file, err := os.OpenFile("test.mp4", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		t.Errorf("os.Openfile")
	}
	defer file.Close()

	f, err := os.Open(".tmp/init.m4s")
	if err != nil {
		t.Errorf("os.Open")
	}
	defer f.Close()

	_, err = io.Copy(file, f)
	if err != nil {
		t.Errorf("io.Copy")
	}

	for i := range n {
		sf, err := os.Open(fmt.Sprintf(".tmp/%d.m4v", i))
		if err != nil {
			t.Errorf("os.Open")
		}

		_, err = io.Copy(file, sf)
		if err != nil {
			t.Errorf("io.Copy")
		}

		log.Printf("Combine segment %d / %d", i, n)
	}
}
