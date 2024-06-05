package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/antchfx/xmlquery"
)

type VideoData struct {
	VideoNo int    `json:"videoNo"`
	Title   string `json:"videoTitle"`
	Date    string `json:"publishDate"`
}

type VideoDataRes struct {
	Code    int       `json:"code"`
	Content VideoData `json:"content"`
}

func getVideoInfo(videoNo int) (*VideoData, error) {
	url := fmt.Sprintf("https://api.chzzk.naver.com/service/v2/videos/%d", videoNo)
	res, err := Get(url)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(res.Body)	
	if err != nil {
		return nil, err
	}

	data := VideoDataRes{}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	return &data.Content, nil
}

type VideoListContent struct {
	Data       []VideoData `json:"data"`
	TotalPages int         `json:"totalPages"`
}

type VideoListRes struct {
	Code    int              `json:"code"`
	Content VideoListContent `json:"content"`
}

func getVideoList(channelId string) ([]VideoData, error) {
	totalPages := 1
	videoList := []VideoData{}
	for page := 0; page < totalPages; page++ {
		url := fmt.Sprintf("https://api.chzzk.naver.com/service/v1/channels/%s/videos?page=%d", channelId, page)
		res, err := Get(url)
		if err != nil {
			return nil, err
		}

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		data := VideoListRes{}
		err = json.Unmarshal(bytes, &data)
		if err != nil {
			return nil, err
		}
		videoList = append(videoList, data.Content.Data...)
		totalPages = data.Content.TotalPages
	}

	return videoList, nil
}

type Video struct {
	Code    int          `json:"code"`
	Content VideoContent `json:"content"`
}

type VideoContent struct {
	Adult   bool   `json:"adult"`
	InKey   string `json:"inKey"`
	VideoId string `json:"videoId"`
}

func getDashUrl(videoNo int) (string, error) {
	url := fmt.Sprintf("https://api.chzzk.naver.com/service/v2/videos/%d", videoNo)
	res, err := Get(url)
	if err != nil {
		return "", err
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	video := Video{}
	err = json.Unmarshal(bytes, &video)
	if err != nil {
		return "", err
	}

	dashUrl := fmt.Sprintf("https://apis.naver.com/neonplayer/vodplay/v1/playback/%s?key=%s", video.Content.VideoId, video.Content.InKey)
	return dashUrl, nil
}

func getVideoUrl(dashUrl string) (string, error) {
	req, err := http.NewRequest("GET", dashUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/xml")
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	doc, err := xmlquery.Parse(res.Body)
	if err != nil {
		return "", err
	}

	representations := xmlquery.Find(doc, "//Representation[@mimeType='video/mp4']")
	maxBandWidth := 0

	for _, node := range representations {
		for _, attr := range node.Attr {
			if attr.Name.Local == "bandwidth" {
				bandwidth, err := strconv.Atoi(attr.Value)
				if err != nil {
					continue
				}

				if bandwidth > maxBandWidth {
					maxBandWidth = bandwidth
				}
			}
		}
	}

	query := fmt.Sprintf("//Representation[@mimeType='video/mp4'][@bandwidth='%d']/BaseURL", maxBandWidth)
	node := xmlquery.FindOne(doc, query)
	if node == nil {
		return "", errors.New("baseurl not found")
	}
	videoUrl := node.InnerText()
	return videoUrl, nil
}

func Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1")

	return http.DefaultClient.Do(req)
}
