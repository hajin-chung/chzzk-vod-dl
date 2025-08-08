FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o cvdl .

FROM alpine:3.18

RUN apk add --no-cache axel tzdata cronie
RUN mkdir -p /usr/local/bin

ENV TZ=Asia/Seoul
RUN cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

COPY --from=builder /app/cvdl /usr/local/bin/cvdl
RUN chmod +x /usr/local/bin/cvdl

COPY job.sh /usr/local/bin/job.sh
RUN chmod +x /usr/local/bin/job.sh

COPY cvdl-crontab /var/spool/cron/crontabs/root
RUN chmod 600 /var/spool/cron/crontabs/root

CMD ["crond", "-f"]
