mod api;
mod memo;
mod utils;
type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;

#[tokio::main]
async fn main() {
    let args: Vec<String> = std::env::args().collect();
    if args.len() != 3 {
        print_help();
        return;
    }
    let mut cvdl = CVDL::new().await;

    let res = match args[1].as_str() {
        "info" => cvdl.handle_info(&args[2]).await,
        "list" => cvdl.handle_list(&args[2]).await,
        "download" => cvdl.handle_download(&args[2]).await,
        "all" => cvdl.handle_all(&args[2]).await,
        _ => Ok(print_help()),
    };

    match res {
        Err(e) => println!("error {e}"),
        _ => (),
    }
}

fn print_help() {
    println!("cvd [Chzzk VOD Downloader]");
    println!("Usage:");
    println!("  cvd list <channel id>");
    println!("  cvd info <video_num>");
    println!("  cvd download <video_num>");
    println!("  cvd all <channel id>");
}

impl std::fmt::Display for api::VideoData {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        write!(f, "No: {}\n", self.num)?;
        write!(f, "Title: {}\n", self.title)?;
        write!(f, "Date: {}\n", self.date)
    }
}

struct CVDL {
    api_client: api::ChzzkClient,
    memo: memo::Memo,
}

impl CVDL {
    async fn new() -> Self {
        let api_client = api::ChzzkClient::new().await.unwrap();
        let memo = memo::Memo::open().await.unwrap();
        CVDL { api_client, memo }
    }

    async fn handle_info(&self, video_num: &String) -> Result<()> {
        let video_info = self.api_client.get_video_info(video_num).await?;
        println!("video info [{video_num}]");
        println!("{video_info}");
        Ok(())
    }

    async fn handle_list(&self, channel_id: &String) -> Result<()> {
        let video_list = self.api_client.get_video_list(channel_id).await?;
        println!("videos in channel {channel_id}");
        video_list.iter().rev().enumerate().for_each(|(i, video)| {
            println!("video {i}");
            println!("{video}");
        });
        Ok(())
    }

    async fn handle_download(&mut self, video_num: &String) -> Result<()> {
        println!("Download start {video_num}");
        let video_info = self.api_client.get_video_info(video_num).await?;
        println!("video info [{video_num}]");
        println!("{video_info}");

        let video_date = utils::format_date(video_info.date.clone())
            .map_err(|e| format!("error on format_date({}): {e}", video_info.date))?;
        let output_path =
            utils::sanitize_filename(format!("{} {}.mp4", video_date, video_info.title));

        let video_url = self.api_client.get_video_url(video_num).await?;
        println!("download {output_path}\nfrom {}", video_url.url);
        match video_url.ty {
            api::VideoURLType::HLS => download_hls(video_url.url, output_path).await?,
            api::VideoURLType::DASH => download_dash(video_url.url, output_path).await?,
        }

        // add video_num to memo
        self.memo.insert(video_info.num).await?;
        Ok(())
    }

    async fn handle_all(&mut self, channel_id: &String) -> Result<()> {
        let video_list = self.api_client.get_video_list(channel_id).await?;
        for video in video_list {
            if !self.memo.check(video.num) {
                let video_num = format!("{}", video.num).to_string();
                self.handle_download(&video_num).await?;
            }
        }
        Ok(())
    }
}

async fn download_hls(hls_url: String, output_path: String) -> Result<()> {
    println!("hls url: {hls_url}");
    let mut ffmpeg_process = tokio::process::Command::new("ffmpeg")
        .arg("-y")
        .args(&["-i", hls_url.as_str()])
        .args(&["-c", "copy"])
        .arg(output_path)
        .spawn()
        .expect("failed to spawn ffmpeg");
    let status = ffmpeg_process.wait().await?;
    println!("ffmpeg exited with status {status}");
    Ok(())
}

async fn download_dash(dash_url: String, output_path: String) -> Result<()> {
    println!("dash url: {dash_url}");
    let mut axel_process = tokio::process::Command::new("axel")
        .args(&["-n", "8", "-o", output_path.as_str(), dash_url.as_str()])
        .spawn()
        .expect("failed to spawn axel");
    let status = axel_process.wait().await?;
    println!("axel exited with status {status}");
    Ok(())
}
