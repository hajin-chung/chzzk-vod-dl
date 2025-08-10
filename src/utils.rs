use chrono::prelude::*;

type Result<T> = super::Result<T>;

pub fn format_date(date: String) -> Result<String> {
    let dt = NaiveDate::parse_from_str(&date, "%Y-%m-%d %H:%M:%S")?;
    Ok(dt.format("%Y.%m.%d").to_string())
}

const FORBIDDEN: [&str; 9] = ["\\", "/", ":", "*", "?", "\"", "<", ">", "|"];

pub fn sanitize_filename(name: String) -> String {
    let mut sanitized = name.clone();

    FORBIDDEN
        .iter()
        .for_each(|c| sanitized = sanitized.replace(c, ""));
    sanitized
}
