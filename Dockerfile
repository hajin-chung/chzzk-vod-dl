# ===== Stage 1: Build the Go binary =====
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o cvd .

# ===== Stage 2: Create the runtime image =====
FROM alpine:3.18

RUN apk add --no-cache axel tzdata cron bash

ENV TZ=Asia/Seoul
RUN cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
COPY --from=builder /app/myapp /usr/local/bin/myapp

RUN chmod +x /usr/local/bin/myapp
WORKDIR /app
RUN mkdir etc vod
RUN echo "0 3 * * * /usr/local/bin/myapp >> /var/log/myapp.log 2>&1" > /etc/crontabs/root
VOLUME /var/log

CMD ["crond", "-f", "-l", "2"]
