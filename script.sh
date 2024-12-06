docker run -d -t \
  --name cvdaemon \
  -v /mnt/d/vod:/app/vod \
  -v /mnt/d/etc:/app/etc \
    cvdaemon:latest
