package downloader

import (
	"errors"
	"fmt"

	"deps.me/chzzk-vod-dl/internal/api"
	"deps.me/chzzk-vod-dl/internal/utils"
)

func DownloadVideo(client *api.ChzzkClient, videoNo int) (*api.VideoData, error) {
	info, err := client.GetVideoInfo(videoNo)
	if err != nil {
		return nil, err
	}
	date, err := utils.FormatDate(info.Date)
	if err != nil {
		return nil, err
	}
	outputName := utils.SanitizeFileName(fmt.Sprintf("%s %s [%d].mp4", date, info.Title, info.VideoNo))

	videoUrl, err := client.GetVideoUrl(videoNo)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[%s] %s\n%s\n", videoUrl.Type, videoUrl.Url, outputName)

	switch videoUrl.Type {
	case api.HLS:
		if err := DownloadHLSVideo(client, videoUrl.Url, outputName); err != nil {
			return nil, err
		}
	case api.DASH:
		if err := DownloadDASHVideo(videoUrl.Url, outputName); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("video type neither hls nor dash")
	}

	return info, nil
}

