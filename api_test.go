package main

import (
	"testing"
	"fmt"
	"log"
	"io"
	"encoding/json"
)

const TestChannelId = "a02dc370efd2befeac97881dc83f11bb"

func TestGetVideoList(t *testing.T) {
	LoadEnv()
	err := LoadSession()
	if err != nil {
		t.Errorf(`LoadSession() error: %s`, err)
		return
	}
	log.Printf("LoadSession() success")

	videoList, err := GetVideoList(TestChannelId)
	if err != nil {
		t.Errorf(`GetVideoList("%s") error: %s`, TestChannelId, err)
		return
	}

	for i, videoData := range videoList {
		fmt.Printf("%02d [%s] %s\n", i, videoData.Date, videoData.Title[:min(50, len(videoData.Title))])
	}
}

// func TestGetDashUrl(t *testing.T) {
// 	LoadEnv()
// 	err := LoadSession()
// 	if err != nil {
// 		t.Errorf(`LoadSession() error: %s`, err)
// 	}
// 	log.Printf("LoadSession() success")
//
// 	videoList, err := GetVideoList(TestChannelId)
// 	if err != nil {
// 		t.Errorf(`GetVideoList("%s") error: %s`, TestChannelId, err)
// 	}
// 	videoListLen := len(videoList)
// 	log.Printf("got video list of length: %d", videoListLen)
//
// 	for _, videoData := range videoList {
// 		dashUrl, err := GetDashUrl(videoData.VideoNo)
// 		if err != nil {
// 			t.Errorf(`GetDashUrl("%d") error: %s`, videoData.VideoNo, err)
// 		}
// 		log.Printf("video [%s] %s dash url\n%s\n", videoData.Date, videoData.Title, dashUrl)
//
// 		videoUrl, err := GetVideoUrl(dashUrl)
// 		if err != nil {
// 			t.Errorf(`GetVideoUrl("%s") error: %s`, dashUrl, err)
// 		}
// 		log.Printf("video [%s] %s url\n%s\n", videoData.Date, videoData.Title, videoUrl)
// 	}
// }

func TestDashHlsV3(t *testing.T) {
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

	for _, videoData := range videoList {
		res, err := Get(fmt.Sprintf("https://api.chzzk.naver.com/service/v3/videos/%d", videoData.VideoNo))
		if err != nil {
			t.Errorf("%d Error on Get v3/videos: %s\n", videoData.VideoNo, err)
		}

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("%d Error on ReadAll res.Body: %s\n", videoData.VideoNo, err)
		}

		video := Video{}
		err = json.Unmarshal(bytes, &video)
		if err != nil {
			t.Errorf("%d Error on Unmarshal bytes: %s\nBytes content: %s\n", videoData.VideoNo, err, string(bytes[:]))
		}
		log.Printf("%d: %+v\n", videoData.VideoNo, video)
	}
}
