docker run -d \
  --name cvdaemon \
  -v /mnt/d/vod:/app/vod \
  -v /mnt/etc:/app/etc \
    cvdaemon:latest
