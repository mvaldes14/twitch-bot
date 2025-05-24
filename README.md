# Twitch Bot

## Description

A comprehensive Go-based Twitch bot designed for streamers who want to enhance their channel with chat commands, music integration, and automated notifications. This bot provides chat interaction features, Spotify playlist control, Discord notifications, and external automation integrations to create an engaging streaming experience.

## Features

### Chat Commands
- `!github` - Links to GitHub profile
- `!dotfiles` - Links to dotfiles repository
- `!song` - Shows currently playing Spotify track
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
- **Channel Point Rewards**: Handles "Next Song", "Add Song", and "Reset Playlist" rewards

### Integrations
- **Spotify**: Music playback control, playlist management, and "Now Playing" display
- **Discord**: Stream notifications when going live
- **External Automation**: Webhooks to automate.mvaldes.dev for additional notifications

## API Endpoints

### Health & Monitoring
*   `/health`: Returns server health status
*   `/metrics`: Prometheus metrics endpoint

### Twitch EventSub Webhooks
*   `/events/chat`: Handles chat messages and commands
*   `/events/follow`: Processes new follower events
*   `/events/subscription`: Processes subscription events
*   `/events/cheer`: Processes cheer/bits events
*   `/events/reward`: Processes channel point reward redemptions

### Subscription Management
*   `/subscriptions`:
    *   `GET`: Lists current EventSub subscriptions
    *   `POST`: Creates new subscription (types: `chat`, `follow`, `subscription`, `cheer`, `reward`, `stream`)
    *   `DELETE`: Deletes all subscriptions (Admin-protected)

### Stream Management
*   `/stream`: Triggers stream live notifications to Discord and external services (Admin-protected)
*   `/test`: Sends test chat message and skips to next Spotify song

### Music Integration
*   `/playing`: Shows currently playing Spotify song with album art
*   `/playlist`: Displays current Spotify playlist

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

#### External Services (Optional)
- `SPOTIFY_TOKEN`: Spotify API token for music integration
- `DISCORD_WEBHOOK_URL`: Discord webhook for stream notifications
- `DOPPLER_TOKEN`: Doppler token for secret management

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
*   `pkgs/discord`: Manages Discord webhook notifications.
*   `pkgs/routes`: Defines HTTP routes and handlers.
*   `pkgs/secrets`: Handles secrets management and Doppler integration.
*   `pkgs/server`: Contains the HTTP server implementation.
*   `pkgs/spotify`: Integrates with Spotify API for music control.
*   `pkgs/subscriptions`: Manages Twitch EventSub subscriptions.
*   `pkgs/telemetry`: Provides logging and Prometheus metrics.
*   `templates`: Stores HTML templates for the web interface.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](file:///home/mvaldes/git/twitch-bot/LICENSE) file for details.

## Credits

- Built with Go 1.22+
- Uses Prometheus for metrics
- Integrates with Twitch EventSub API
- Supports Spotify Web API
- Uses Doppler for secret management

## Source Code

Repository: https://github.com/mvaldes14/twitch-bot

See it live at https://links.mvaldes.dev/stream
