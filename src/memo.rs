use std::collections::HashSet;
use tokio::{fs, io::AsyncWriteExt};

type Result<T> = super::Result<T>;

pub struct Memo {
    pub memo_path: String,
    pub numbers: HashSet<u32>,
}

impl Memo {
    pub async fn open() -> Result<Self> {
        let memo_path = std::env::var("CVDL_SESSION").unwrap_or("memo.dat".to_string());
        let memo_bytes = fs::read(&memo_path).await?;
        let memo_str = String::from_utf8(memo_bytes)?;
        let mut numbers: HashSet<u32> = HashSet::new();
        memo_str.split("\n").for_each(|s| {
            numbers.insert(s.parse::<u32>().unwrap_or(0));
        });
        Ok(Memo { memo_path, numbers })
    }

    pub fn check(&self, video_num: u32) -> bool {
        self.numbers.contains(&video_num)
    }

    pub async fn insert(&mut self, video_num: u32) -> Result<()> {
        let mut memo_file = fs::OpenOptions::new()
            .append(true)
            .create(true)
            .open(&self.memo_path)
            .await?;
        memo_file
            .write_all(format!("{video_num}").as_bytes())
            .await?;
        memo_file.write_all(b"\n").await?;
        self.numbers.insert(video_num.clone());
        Ok(())
    }
}
