# responder-cli

`responder` is a command-line testing harness for prepared911
features, starting with the features behind the portal's
Responders tab. It reproduces the Audio Demo flow from a terminal:
`responder run-demo` creates an Incident from an Audio Demo with
sensible defaults, using the same API the portal uses, then prints
the incident ID and a clickable portal URL. `responder start-app`
brings up the local prepared911 API stack and waits until it is
healthy.

One binary, subcommands for each testing task, so new commands can
be added later without new tools.

## Prerequisites

- Go 1.27+
- A prepared911 checkout (defaults to `~/workspace/prepared911`)
  with its own prerequisites for `start-app` (OrbStack,
  1Password-provided secrets, AWS credentials) and the `just`
  command on `PATH`
- Membership in a dispatch center with the audio demo setting
  enabled (for example Engineering PD)

## Build

```sh
go build -o responder ./cmd/responder
```

## Install

```sh
go install ./cmd/responder
```

This puts `responder` on `PATH` (under `$(go env GOPATH)/bin`).

## Example

```sh
# Bring up the local API stack (once per machine boot).
responder start-app

# Create an incident from the default call type.
responder run-demo
# Incident created: abc123
# Portal URL: http://localhost:3002/chatroom/abc123

# Try a different call type at a fixed location.
responder run-demo --call-type "House Fire - English" \
  --lat 40.8017396 --lng -73.7379377
```

## Configuration

Precedence is flag, then environment variable, then default.

| Setting             | Flag              | Env                    | Default                          |
| ------------------- | ----------------- | ---------------------- | -------------------------------- |
| Caller phone        | `--phone`         | `RESPONDER_CALLER_PHONE` | `+1 (817) 973-1331`            |
| Country             | `--country`       | `RESPONDER_COUNTRY`      | `US`                           |
| Call type           | `--call-type`     | `RESPONDER_CALL_TYPE`    | `Shooting Incident - English`  |
| Latitude            | `--lat`           | `RESPONDER_LAT`          | looked up from your IP         |
| Longitude           | `--lng`           | `RESPONDER_LNG`          | looked up from your IP         |
| API base URL        | `--api-url`       | `RESPONDER_API_URL`      | `http://127.0.0.1:3000`        |
| Portal base URL     | `--portal-url`    | `RESPONDER_PORTAL_URL`   | `http://localhost:3002`        |
| Auth token          | `--token`         | `RESPONDER_TOKEN`        | dev token in `.env.local`      |
| prepared911 checkout| `--prepared911-dir`| `PREPARED911_DIR`       | `~/workspace/prepared911`      |
| User email          | `--user-email`    | `RESPONDER_USER_EMAIL`   | `git config user.email`        |

Phone numbers are sent to the API in E.164 regardless of how they
are typed. The country only supplies the default region for numbers
entered without a `+` (a `+1` prefix for US).

Location defaults to your machine's approximate location looked up
from ip-api.com, mirroring the portal's "Use My Location" button.
Pass `--lat` and `--lng` together to pick a location explicitly or
to work offline; a failed lookup is an error, never a silent
fallback. The demo is always created with custom coordinates,
location production on, CAD incident production on, and
forced-abandoned off, matching the portal's defaults.

The auth token resolves from `--token`/`RESPONDER_TOKEN`, then the
`GRAPHIQL_DEV_TOKEN` in the checkout's `.env.local`, then fails with
instructions for minting one (`just api graphiql-token` in the
prepared911 checkout).

`responder --help` and per-command help describe every flag and
environment variable.

## Call types

Only the always-on built-in demos are supported; their audio is
copied from the dispatch UI's built-in demo list and kept in sync by
hand. Custom (database-backed) demos are not supported. An unknown
`--call-type` fails listing the valid names.

## Tests

```sh
go test ./...
```

#Todo:
[] `login` command - re-uses whatever api and dispatch use, ex: lets a user choose google
