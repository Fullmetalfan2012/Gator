# Gator

Gator is a command-line RSS feed aggregator written in Go. It lets multiple users register, follow feeds, aggregate posts from those feeds into a Postgres database, and browse them from the terminal.

## Prerequisites

You need the following installed on your machine to run Gator:

- **Go** (1.26 or newer) — https://go.dev/doc/install
- **PostgreSQL** (15 or newer) — https://www.postgresql.org/download/

Make sure you have a running Postgres server and a database that Gator can connect to. You will need the connection string for that database when you set up the config file below.

## Install

Install the `gator` CLI with `go install`:

```bash
go install github.com/Fullmetalfan2012/Gator@latest
```

This builds and installs a `Gator` binary into your `$GOBIN` (usually `~/go/bin`). Make sure that directory is on your `PATH` so you can run the command from anywhere.

## Configuration

Gator reads its configuration from `~/.gatorconfig.json`. Create the file by hand with this structure:

```json
{
  "db_url": "postgres://username:password@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

- `db_url` — the Postgres connection string for the database you want Gator to use.
- `current_user_name` — leave empty; it gets set automatically when you register or log in.

Before running any commands, apply the database schema from `sql/schema/` to your Postgres database (for example with [goose](https://github.com/pressly/goose)).

## Usage

Once the CLI is installed and the config file is in place, run commands with `gator <command> [args...]`:

```bash
gator register <name>          # create a new user and log them in
gator login <name>             # switch the active user
gator users                    # list all users, marking the current one
gator addfeed <name> <url>     # add a feed and follow it as the current user
gator feeds                    # list every feed in the database
gator follow <url>             # follow an existing feed
gator following                # show feeds the current user follows
gator unfollow <url>           # stop following a feed
gator agg <duration>           # continuously fetch feeds on an interval (e.g. 1m, 30s)
gator browse [limit]           # show recent posts from followed feeds (default 2)
gator reset                    # wipe all users (useful for development)
```

Typical first-time flow:

```bash
gator register alice
gator addfeed "Hacker News" https://news.ycombinator.com/rss
gator agg 1m          # leave this running in one terminal
gator browse 10       # in another terminal, read the latest posts
```

## Development

`go run .` is fine while you're iterating on the code, but for normal use install the binary with `go install` and run `gator` directly.
