# Twitch Bot

## Description

A Go-based Twitch bot designed for streamers who want to enhance their channel with chat commands and automated notifications. This bot provides chat interaction features, Twitch EventSub handlers, Discord notifications, and external automation integrations to create an engaging streaming experience.

## Features

### Chat Commands
- `!github` - Links to GitHub profile
- `!dotfiles` - Links to dotfiles repository
- `!social` - Shows social media links
- `!blog` - Links to blog
- `!youtube` - Links to YouTube channel
- `!discord` - Links to Discord server
- `!commands` - Lists available commands
- `!today <title>` - Updates stream title/game (streamer only)

### Twitch Event Responses
- **Follows**: Sends "Gracias por el follow" message
- **Subscriptions**: Sends "Gracias por el sub" message  
- **Cheers/Bits**: Sends "Gracias por los bits" message

### Integrations
- **Discord**: Stream notifications when going live
- **External Automation**: Webhooks to automate.mvaldes.dev for additional notifications

## API Endpoints

### Health & Monitoring
*   `/health`: Returns server health status
*   `/metrics`: Prometheus metrics endpoint

### Twitch EventSub Webhooks
*   `/chat`: Handles chat messages and commands
*   `/follow`: Processes new follower events
*   `/sub`: Processes subscription events
*   `/cheer`: Processes cheer/bits events
*   `/stream-online`: Processes stream online events
*   `/stream-offline`: Processes stream offline events

### Subscription Management
Admin-protected API routes live under `/api`:
*   `GET /api/list`: Lists current EventSub subscriptions
*   `POST /api/create`: Creates a new subscription (types: `chat`, `follow`, `subscription`, `cheer`, `streamon`, `streamoff`)
*   `POST /api/delete`: Deletes all subscriptions

### Stream Management
*   `/stream-online`: Triggers stream live notifications to Discord and external services
*   `/stream-offline`: Records stream end timing
*   `/test`: Sends a test notification

## Setup Instructions

To set up the Twitch Bot, follow these steps:

### Prerequisites

*   Go 1.22 or higher

### Dependencies

The project uses Go modules for dependency management. To install the dependencies, run:

```bash
go mod download
go mod vendor
```

### Configuration

The bot uses environment variables for configuration. Required environment variables:

#### Twitch API
- `TWITCH_TOKEN`: Twitch app access token
- `TWITCH_CLIENT_ID`: Twitch application client ID
- `TWITCH_CLIENT_SECRET`: Twitch application client secret
- `TWITCH_USER_TOKEN`: Twitch user token for chat and API access
- `TWITCH_REFRESH_TOKEN`: Refresh token for token renewal

#### Notifications
- `DISCORD_WEBHOOK`: Discord webhook URL for stream notifications
- `GOTIFY_APPLICATION_TOKEN`: Gotify application token for push notifications

#### Redis
- `REDIS_URL`: Redis connection address (e.g. `localhost:6379`) used for token caching and storage

#### OpenTelemetry
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP HTTP endpoint for exporting traces and metrics (required to enable observability)
- `OTEL_SERVICE_VERSION`: Service version reported in telemetry resource attributes (defaults to `1.0.0`)
- `OTEL_ENVIRONMENT`: Deployment environment name, e.g. `development` or `production` (defaults to `development`)
- `OTEL_INSECURE_ENDPOINT`: Set to `true` to disable TLS verification for the OTLP endpoint (defaults to `false`)

#### Other
- `ADMIN_TOKEN`: Token used to authenticate admin-protected API routes
- `DOPPLER_TOKEN`: Doppler token for secret management (optional)

#### Development
The project uses Nix flakes for development environment. Run `direnv allow` to load the environment.

## Usage Instructions

To run the bot:

```bash
go run main.go
```

The server will start listening on port 3000. You can then access it via your web browser or other HTTP clients.

## Project Structure

The project is organized into several packages:

*   [`main.go`](file:///home/mvaldes/git/twitch-bot/main.go): The main entry point of the application.
*   `pkgs/actions`: Handles bot commands and chat interactions.
*   `pkgs/notifications`: Manages Discord and Gotify notifications.
*   `pkgs/routes`: Defines HTTP routes and handlers.
*   `pkgs/secrets`: Handles secrets management and Doppler integration.
*   `pkgs/server`: Contains the HTTP server implementation.
*   `pkgs/subscriptions`: Manages Twitch EventSub subscriptions.
*   `pkgs/telemetry`: Provides logging, OpenTelemetry tracing, and metrics.
*   `pkgs/cache`: Redis-based token caching and storage.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](file:///home/mvaldes/git/twitch-bot/LICENSE) file for details.

## Credits

- Built with Go 1.22+
- Uses OpenTelemetry for tracing and metrics
- Integrates with Twitch EventSub API
- Uses Redis for token caching
- Uses Doppler for secret management

## Source Code

Repository: https://github.com/mvaldes14/twitch-bot

See it live at https://links.mvaldes.dev/stream
