FROM golang:1.24.6-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o cvdl .

FROM debian:bookworm
WORKDIR /app

RUN mkdir -p /usr/local/bin

# install dependencies
RUN apt update
RUN apt install -y axel cron wget xz-utils

RUN wget -O ffmpeg.tar.xz https://github.com/BtbN/FFmpeg-Builds/releases/download/autobuild-2024-08-31-12-50/ffmpeg-n5.1.6-2-g0e8b267a97-linux64-lgpl-5.1.tar.xz
RUN mkdir ffmpeg
RUN tar -xf ffmpeg.tar.xz -C ffmpeg --strip-components=1
RUN cp ffmpeg/bin/* /usr/local/bin/

# set timezone
ENV TZ=Asia/Seoul
RUN cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

COPY --from=builder /app/cvdl /usr/local/bin/cvdl
RUN chmod +x /usr/local/bin/cvdl

COPY job.sh /usr/local/bin/job.sh
RUN chmod +x /usr/local/bin/job.sh

COPY cvdl-crontab /etc/cron.d/cvdl-crontab
RUN chmod 0644 /etc/cron.d/cvdl-crontab

RUN touch /var/log/cvdl-cron.log

CMD echo "cvdl started" >> /var/log/cvdl-cron.log
CMD cron -f && tail -f /var/log/cvdl-cron.log
