# chzzk-vod-dl

cld : Chzzk VOD Downloader

```bash
$ cvd
cvd [Chzzk VOD Downloader]
Usage:
  cvd list <channel id>
  cvd info <video no>
  cvd download <video no>
  cvd all <channel id>
```

## TODO

- [x] download video by video number
- [x] record of already downloaded videos
- [x] migrate to v3
- [x] download hls
- [x] rewrite it in rust
- [ ] better error handling
- [ ] better logging
- [ ] faster download fmp4 fragments (.m4v): download fragments concurrently considering max bandwidth or with using axel
