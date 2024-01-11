FROM golang:1.20-alpine

# Install Doppler CLI
# Install Doppler CLI
RUN wget -q -t3 'https://packages.doppler.com/public/cli/rsa.8004D9FF50437357.key' -O /etc/apk/keys/cli@doppler-8004D9FF50437357.rsa.pub && \
    echo 'https://packages.doppler.com/public/cli/alpine/any-version/main' | tee -a /etc/apk/repositories && \
    apk add doppler

WORKDIR /app

COPY . /app

# Build the Go app
RUN go build -o twitch-bot .

ENTRYPOINT ["doppler", "run", "--", "/app/twitch-bot"]
