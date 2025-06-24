FROM golang:1.22-alpine

WORKDIR /app

COPY . /app

# Build the Go app
RUN go build -o twitch-bot

ENTRYPOINT ["/app/twitch-bot"]
