FROM golang:1.23-bookworm
WORKDIR /app

COPY ./*.go ./
COPY ./go.* ./

RUN go build -o cvd . && ls -al
RUN apt-get update && apt-get install -y axel

ENTRYPOINT ["./cvd"]
