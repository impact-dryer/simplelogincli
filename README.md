# simplelogincli

A tiny Go CLI to create SimpleLogin email aliases using the API described in `api.md`.

Features:
- Store and use your API key securely
- Create random aliases (uuid or word mode)
- Create custom aliases (prefix + signed suffix), with interactive suffix picker
- Inspect available suffix options
- Show account info (`whoami`)

## Requirements
- Go 1.22+ (to build from source)
- A SimpleLogin API key

## Build
Project structure was refactored to use packages and a proper entrypoint under `cmd/simplelogin`.

```zsh
cd simplelogincli
# build the CLI binary from the new entrypoint
go build -o simplelogin ./cmd/simplelogin
```

## Configuration
The CLI looks for configuration in this order:
1) Environment variables (highest precedence)
2) Config file at `$XDG_CONFIG_HOME/simplelogincli/config.json` or `~/.config/simplelogincli/config.json`

Environment variables:
- `SIMPLELOGIN_API_KEY` — API key
- `SIMPLELOGIN_BASE_URL` — Base URL (default: `https://app.simplelogin.io`)

You can also save the API key into the config file using the CLI:
```zsh
./simplelogin set-key --api-key "<your_api_key>" [--base-url https://app.simplelogin.io]
```

## Usage
```zsh
./simplelogin help
```

### Show account info
```zsh
./simplelogin whoami
# or override saved key
SIMPLELOGIN_API_KEY=... ./simplelogin whoami
```

### List alias options (suffixes, prefix suggestion)
```zsh
./simplelogin options --hostname example.com
```

### Create a random alias
```zsh
# Use your default settings on the server
./simplelogin random

# Force mode (uuid or word) and attach a note
./simplelogin random --mode word --note "For example.com"

# Include a hostname to help suggestions/history
./simplelogin random --hostname example.com
```
The command prints the newly created alias email to stdout on success.

### Delete alias
```zsh
./simplelogin --delete --email "<email_to_delete>"
```


### Create a custom alias (prefix + suffix)
You can either provide the signed suffix directly (from `options`) or pick it by the plain suffix value.

- Non-interactive, by plain suffix:
```zsh
./simplelogin custom \
  --prefix "myshop" \
  --suffix ".yeah@sl.lan" \
  --note "Shop account"
```
The CLI fetches options, matches `--suffix`, and uses the associated `signed_suffix`.

- Non-interactive, by signed suffix:
```zsh
./simplelogin custom \
  --prefix "myshop" \
  --signed-suffix ".yeah@sl.lan.X6_7OQ.i8XL4xsMsn7dxDEWU8eF-Zap0qo"
```

- Interactive suffix selection:
```zsh
./simplelogin custom --prefix "myshop"
# The CLI will list suffixes and ask you to choose.
```

- Specify mailbox owners for the alias (defaults to your default mailbox if omitted):
```zsh
./simplelogin custom --prefix "work" --suffix ".yeah@sl.lan" --mailbox-ids "1,2"
```

The command prints the newly created alias email to stdout on success.

## Tests
Unit tests cover the configuration layer and API client behavior using `httptest`.

```zsh
# from project root
go test ./...
```

### Integration tests (optional)
There are integration tests (behind the `integration` build tag) that hit the real API. They will:
- Validate your API key with `/api/user_info`
- Create a random alias and delete it
- Create a custom alias and delete it (if suffixes are available)

Run them only when you set a valid API key (and optional base URL):
```zsh
# Be careful: this will create and then delete aliases in your account
SIMPLELOGIN_API_KEY=your_key \
SIMPLELOGIN_BASE_URL=https://app.simplelogin.io \
  go test -tags=integration ./pkg/api -v
```
If env vars are not set, integration tests are skipped.

## Notes and troubleshooting
- 401 Unauthorized: check your API key with `./simplelogin whoami` or re-run `set-key`.
- `can_create` false or quota exceeded: the API returns an error message; the CLI prints it to stderr.
- Premium-only suffixes: trying to create an alias with a premium-only suffix will return a 4xx with an explanatory error.
- Base URL: override with `--base-url` or `SIMPLELOGIN_BASE_URL` to target self-hosted instances.

## Development
Quick smoke test after changes:
```zsh
# build
go build -o simplelogin ./cmd/simplelogin
# help output
./simplelogin help
```

## License
MIT (or the same license as your project).
