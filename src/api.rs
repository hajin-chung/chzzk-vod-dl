use regex::{Regex, RegexBuilder};
use reqwest;
use serde::Deserialize;
use tokio::fs;

const USER_AGENT: &str = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1";
const API_URL: &str = r#"https://api.chzzk.naver.com"#;
const HLS_REGEX: &str = r#"#EXT-X-STREAM-INF:.*?BANDWIDTH=(?<bandwidth>\d+).*$\n(?<url>.*$)"#;
const DASH_REGEX: &str = r#"<Representation.*?bandwidth=\"(?<bandwidth>\d+)\".*?>.*?<BaseURL>(?<url>.*?)<\/BaseURL>.*?<\/Representation>"#;

type Result<T> = super::Result<T>;
pub struct ChzzkClient {
    http_client: reqwest::Client,
    headers: reqwest::header::HeaderMap,
}

#[derive(Deserialize, Debug)]
pub struct VideoData {
    #[serde(rename = "videoNo")]
    pub num: u32,
    #[serde(rename = "videoTitle")]
    pub title: String,
    #[serde(rename = "publishDate")]
    pub date: String,
    // #[serde(rename = "adult")]
    // pub adult: bool,
    #[serde(rename = "inKey")]
    pub in_key: Option<String>,
    #[serde(rename = "liveRewindPlaybackJson")]
    pub playback_json: Option<String>,
    #[serde(rename = "videoId")]
    pub id: String,
}

#[derive(Deserialize)]
pub struct VideoDataRes {
    // code: u32,
    content: VideoData,
}

#[derive(Deserialize, Debug)]
pub struct VideoListContent {
    #[serde(rename = "data")]
    data: Vec<VideoData>,

    #[serde(rename = "totalPages")]
    total_pages: u32,
}

#[derive(Deserialize)]
pub struct VideoListRes {
    // code: u32,
    content: VideoListContent,
}

#[derive(Deserialize)]
pub struct VideoPlaybackMedia {
    path: String,
}

#[derive(Deserialize)]
pub struct VideoPlayback {
    media: Vec<VideoPlaybackMedia>,
}

pub enum VideoURLType {
    HLS,
    DASH,
}

pub struct VideoURL {
    pub ty: VideoURLType,
    pub url: String,
}

impl ChzzkClient {
    pub async fn new() -> Result<Self> {
        let http_client = reqwest::Client::new();
        let mut headers = reqwest::header::HeaderMap::new();
        let session = get_session().await?;
        headers.insert("User-Agent", USER_AGENT.parse()?);
        headers.insert("Cookie", session.as_str().parse()?);

        Ok(ChzzkClient {
            http_client,
            headers,
        })
    }

    pub fn get<U: reqwest::IntoUrl>(&self, url: U) -> reqwest::RequestBuilder {
        self.http_client.get(url).headers(self.headers.clone())
    }

    pub async fn get_video_info(&self, video_num: &String) -> Result<VideoData> {
        let url = format!("{API_URL}/service/v3/videos/{video_num}");
        let video_data_res = self.get(url).send().await?.json::<VideoDataRes>().await?;
        Ok(video_data_res.content)
    }

    pub async fn get_video_list(&self, channel_id: &String) -> Result<Vec<VideoData>> {
        let mut total_pages: u32 = 1;
        let mut video_list: Vec<VideoData> = Vec::new();

        let mut page = 0;
        loop {
            if page == total_pages {
                break;
            }

            let url = format!("{API_URL}/service/v1/channels/{channel_id}/videos?page={page}");
            let mut video_list_res = self
                .http_client
                .get(url)
                .send()
                .await?
                .json::<VideoListRes>()
                .await?;
            video_list.append(&mut video_list_res.content.data);
            total_pages = video_list_res.content.total_pages;

            page += 1;
        }
        Ok(video_list)
    }

    pub async fn get_video_url(&self, video_num: &String) -> Result<VideoURL> {
        let video_info = self.get_video_info(video_num).await?;
        if let Some(_) = video_info.in_key {
            let url = self.get_dash_url(&video_info).await?;
            Ok(VideoURL {
                ty: VideoURLType::DASH,
                url: url,
            })
        } else if let Some(_) = video_info.playback_json {
            let url = self.get_hls_url(&video_info).await?;
            Ok(VideoURL {
                ty: VideoURLType::HLS,
                url: url,
            })
        } else {
            Err("strange json".into())
        }
    }

    async fn get_hls_url(&self, video_data: &VideoData) -> Result<String> {
        let playback_json = video_data.playback_json.as_ref().unwrap();
        let playback_data: VideoPlayback = serde_json::from_str(&playback_json)?;
        let playlist_url_result: Result<String> = match playback_data.media.iter().nth(0) {
            Some(media) => Ok(media.path.clone()),
            None => Err("playback json has no media".into()),
        };
        let playlist_url = url::Url::parse(&playlist_url_result?)?;
        let hls_str = self.get(playlist_url.clone()).send().await?.text().await?;
        println!("{hls_str}");

        let re = RegexBuilder::new(HLS_REGEX)
            .multi_line(true)
            .build()
            .unwrap();
        let target = re
            .captures_iter(&hls_str)
            .map(|c| c.extract::<2>().1)
            .map(|[bandwidth_str, url]| (bandwidth_str.parse::<u32>().unwrap(), url))
            .max()
            .unwrap();
        let path = target.1;
        let target_url = playlist_url.join(path)?;
        Ok(target_url.as_str().into())
    }

    async fn get_dash_url(&self, video_data: &VideoData) -> Result<String> {
        let in_key = video_data.in_key.as_ref().unwrap();
        let dash_url = format!(
            "https://apis.naver.com/neonplayer/vodplay/v1/playback/{}?key={}",
            video_data.id, in_key
        );
        let xml_str = self
            .get(&dash_url)
            .header("Accept", "application/xml")
            .send()
            .await?
            .text()
            .await
            .map_err(|e| format!("error when fetching dash url: {dash_url} {e}"))?;

        let re = Regex::new(DASH_REGEX).unwrap();
        let target: (u32, &str) = re
            .captures_iter(&xml_str)
            .map(|c| c.extract::<2>().1)
            .map(|[bandwidth_str, url]| (bandwidth_str.parse().unwrap(), url))
            .map(|(bandwidth, url)| {
                println!("{bandwidth}: {url}");
                (bandwidth, url)
            })
            .max()
            .unwrap();

        println!("{:#?}", target);
        Ok(target.1.into())
    }
}

async fn get_session() -> Result<String> {
    let session_file_path = std::env::var("CVDL_SESSION").unwrap_or("session.dat".into());
    let session_contents = fs::read(&session_file_path)
        .await
        .map_err(|e| format!("error on read({session_file_path}): {e}"))?;
    let session = String::from_utf8(session_contents)?.trim().to_string();
    Ok(session)
}

#[cfg(test)]
mod tests {
    use super::*;

    // const DASH_VIDEO_NUM : &str = "8094423";
    const HLS_VIDEO_NUM: &str = "8541925";
    const TARGET_NUM: &str = HLS_VIDEO_NUM;

    #[tokio::test]
    async fn test_get_video_info() -> Result<()> {
        let client = ChzzkClient::new().await?;
        let video_info = client.get_video_info(&String::from(TARGET_NUM)).await?;
        println!("{}", video_info);
        println!("{:#?}", video_info);
        Ok(())
    }

    #[tokio::test]
    async fn test_get_video_list() -> Result<()> {
        let client = ChzzkClient::new().await?;
        let video_list = client
            .get_video_list(&String::from("a02dc370efd2befeac97881dc83f11bb"))
            .await?;
        println!("{}", video_list.len());
        // for i in 0..video_list.len() {
        //     let video = &video_list[i];
        //     println!("video {i}");
        //     println!("{video}");
        // }
        Ok(())
    }

    #[tokio::test]
    async fn test_get_video_url() -> Result<()> {
        let client = ChzzkClient::new().await?;
        let video_url = client.get_video_url(&String::from(TARGET_NUM)).await?;
        match video_url.ty {
            VideoURLType::HLS => println!("type: HLS"),
            VideoURLType::DASH => println!("type: DASH"),
        }
        println!("url: {}", video_url.url);
        Ok(())
    }

    #[tokio::test]
    async fn test_chzzk_api() -> Result<()> {
        // let url = format!("{API_URL}/service/v3/videos/8541925");
        let url = format!("{API_URL}/service/v3/videos/{TARGET_NUM}");
        let client = ChzzkClient::new().await?;
        let json: serde_json::Value = client.http_client.get(url).send().await?.json().await?;
        println!("{:#?}", json);
        Ok(())
    }
}
