# Feedie

A terminal RSS/Atom feed reader with a client-server architecture. The server fetches and stores feeds in a local SQLite database; the client is a TUI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) that displays feeds, tags, and entries with optional thumbnail support.

<img src="https://jasco.website/projects/feedie/demo1.png" alt="Demo of Feedie" width=500>

## Features

- RSS and Atom feed parsing via [gofeed](https://github.com/mmcdole/gofeed)
- Tag-based feed organization
- Thumbnail display via Kitty graphics protocol or ueberzug
- Fuzzy filtering of feeds and entries
- Configurable keybindings and color scheme
- Link opening by URL pattern or MIME type
- Link yanking to clipboard
- Background feed refresh (default every ~2.5 hours)

## Requirements

- Go 1.24+
- A terminal that supports true color
- (Optional) Kitty terminal or ueberzug for image thumbnails

## Building

### Server

```sh
cd server
go build -o feedie-server
```

### Client

```sh
cd client
make build      # produces feedie-client
make install    # symlinks to ~/.local/bin/feedie
```

## Running

Start the server first:

```sh
./feedie-server
```

Then launch the client:

```sh
./feedie-client
```

## Configuration

### Server

Configured via environment variables:

| Variable | Default | Description |
|---|---|---|
| `FEEDIE_SERVER_PORT` | `2550` | Port to listen on |
| `FEEDIE_SERVER_REFRESH_RATE` | `9000` | Feed refresh interval in seconds (~2.5 hrs) |
| `FEEDIE_SERVER_DB_PATH` | `~/.local/share/feedie/feedie.db` | SQLite database path |

### Client

Config file is written on first run to `~/.config/feedie/conf.json`. Override the path with `-c <path>` or `--config <path>`.

Key config fields:

| Field | Default | Description |
|---|---|---|
| `server` | `http://localhost` | Server address |
| `port` | `:2550` | Server port |
| `thumbnailbackend` | `kitty` | Image backend: `kitty`, `ueberzug`, or `""` to disable |
| `thumbnailratio` | `0.4` | Fraction of the pane width used for thumbnails |
| `thumbnailpath` | `/tmp/feedie-go` | Directory for cached thumbnails |
| `thumbnailscaler` | `fit_contain` | Scaling mode for images (only for Ueberzug backend)|
| `linkcopycommand` | `xclip -i -selection clipboard` | Command used to yank links |
| `defaultopener` | `xdg-open` | Fallback command for opening links |

`urlopener` and `typeopener` are maps of regex/MIME-type patterns to commands, checked before `defaultopener`.

## CLI Usage

```sh
# Add a feed
feedie --add_feed <url>

# Add a feed and immediately assign it to a tag
feedie --add_feed <url> --tag <tag_name>
```

## Default Keybindings

| Key | Action |
|---|---|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Open selected feed/entry |
| `a` | Add feed |
| `t` | Add tag |
| `T` | Modify tag members |
| `d` | Delete feed or tag |
| `r` | Refresh list |
| `y` | Copy link to clipboard |
| `o` | Open link menu |
| `m` | Feed menu |
| `/` | Filter |
| `g` / `G` | Go to start / end |
| `?` | Toggle help |
| `Tab` | Change focus |
| `Q` | Quit |

All keybindings are rebindable in `conf.json` under the `keys` object.

## Database Migrations

Two one-shot migration commands are available for upgrading an existing database:

```sh
feedie-server migrate_add_link_id   # adds id column to links table
feedie-server migrate_dedup_guid    # removes old-hash duplicate entries
```

## License

GPL-3.0
