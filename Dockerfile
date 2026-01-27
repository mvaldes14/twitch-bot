FROM golang:1.24-alpine

WORKDIR /app

COPY . /app

# Build the Go app
RUN go build -o twitch-bot

ENTRYPOINT ["/app/twitch-bot"]
