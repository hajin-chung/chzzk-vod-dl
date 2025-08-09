package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
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

func GetVideoInfo(videoNo int) (*VideoData, error) {
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

func GetVideoList(channelId string) ([]VideoData, error) {
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

	slices.Reverse(videoList)
	return videoList, nil
}

type Video struct {
	Code    int          `json:"code"`
	Content VideoContent `json:"content"`
}

type VideoContent struct {
	Adult   bool   `json:"adult"`
	InKey   string `json:"inKey"`
	PlaybackJson string `json:"liveRewindPlaybackJson"`
	VideoId string `json:"videoId"`
}
type VideoPlayback struct {
	Media []VideoPlaybackMedia `json:"media"`
}
type VideoPlaybackMedia struct {
	Path string `json:"path"`
}

type VideoType string
const (
	HLS VideoType = "HLS"
	DASH VideoType = "DASH"
)

type VideoUrl struct {
	Type VideoType
	Url string
}

func GetVideoUrl(videoNo int) (*VideoUrl, error) {
	url := fmt.Sprintf("https://api.chzzk.naver.com/service/v3/videos/%d", videoNo)
	res, err := Get(url)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	video := Video{}
	err = json.Unmarshal(bytes, &video)
	if err != nil {
		return nil, err
	}

	if (video.Content.InKey == "") {
		// new hls playback
		playbackData := VideoPlayback{}
		if err := json.Unmarshal([]byte(video.Content.PlaybackJson), &playbackData); err != nil {
			return nil, err
		}
		videoUrl := VideoUrl {
			Url: playbackData.Media[0].Path,
			Type: HLS,
		}
		return &videoUrl, nil
	} else {
		// old dash playback
		dashUrl := fmt.Sprintf("https://apis.naver.com/neonplayer/vodplay/v1/playback/%s?key=%s", video.Content.VideoId, video.Content.InKey)

		req, err := http.NewRequest("GET", dashUrl, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Accept", "application/xml")
		req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1")

		res, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		doc, err := xmlquery.Parse(res.Body)
		if err != nil {
			return nil, err
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
			return nil, errors.New("baseurl not found")
		}

		videoUrl := VideoUrl {
			Url: node.InnerText(),
			Type: DASH,
		}
		return &videoUrl, nil
	}
}

func Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1")
	if sessionLoadSuccess {
		req.Header.Add("Cookie", session)
	}

	return http.DefaultClient.Do(req)
}
