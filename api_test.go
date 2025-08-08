package main

import (
	"testing"
	"fmt"
	"log"
)

const TestChannelId = "a02dc370efd2befeac97881dc83f11bb"

func TestGetVideoList(t *testing.T) {
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

func TestGetDashUrl(t *testing.T) {
	err := LoadSession()
	if err != nil {
		t.Errorf(`LoadSession() error: %s`, err)
	}
	log.Printf("LoadSession() success")

	videoList, err := GetVideoList(TestChannelId)
	if err != nil {
		t.Errorf(`GetVideoList("%s") error: %s`, TestChannelId, err)
	}
	log.Printf("got video list of length: %d", len(videoList))

	dashUrl, err := GetDashUrl(videoList[0].VideoNo)
	if err != nil {
		t.Errorf(`GetDashUrl("%s") error: %s`, TestChannelId, err)
	}
	log.Printf("got dash url: %s\n", dashUrl)
}
