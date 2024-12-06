# ===== Stage 1: Build the Go binary =====
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o cvd .

# ===== Stage 2: Create the runtime image =====
FROM alpine:3.18

RUN apk add --no-cache axel tzdata cronie bash

ENV TZ=Asia/Seoul
RUN cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
COPY --from=builder /app/cvd /usr/local/bin/cvd

RUN chmod +x /usr/local/bin/cvd
WORKDIR /app
RUN mkdir etc vod
RUN echo "0 3 * * * /usr/local/bin/cvd >> /var/log/cvd.log 2>&1" > /etc/crontabs/root
VOLUME /var/log

CMD ["crond", "-f", "-l", "2"]
