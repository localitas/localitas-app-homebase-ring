# Homebase Ring Plugin

Ring device integration for Homebase. Controls Ring doorbells, cameras, chimes, and alarm systems.

## Acquiring a Ring Refresh Token

Ring uses OAuth2 with refresh tokens. Generate one using the built-in `auth` subcommand:

```bash
cd ~/localitas-app-homebase-ring
make build
./bin/homebase-ring-server auth
```

This prompts for your Ring email and password. If you have 2FA enabled (recommended), it will ask for the code sent to your phone or authenticator app. On success, it prints the refresh token and the `curl` command in step 1 below with the token filled in, aimed at core on `--core-url` (default `$CORE_URL`, else `http://localhost:8090`).

The refresh token rotates automatically. Homebase-ring writes updated tokens back to Vault, so you only need to do this once.

## Setup

### 1. Store the Token in Vault

After running `auth`, store the refresh token:

```bash
curl -X POST http://localhost:8090/apps/vault/api/credentials \
  -H "Authorization: Bearer $LOCALITAS_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Ring", "data": {"ring_refresh_token": "YOUR_REFRESH_TOKEN_HERE"}}'
```

`LOCALITAS_API_TOKEN` holds your API token. Core's token is `core.auth.api_token` in `~/.localitas/config-core.yaml`; homebase-ring and the Localitas SDKs read it from there automatically when the variable is unset.

Note the `public_id` in the response — you'll need it in step 3.

### 2. Start homebase-ring

```bash
cd ~/localitas-app-homebase-ring && make start
```

`make start` builds `bin/homebase-ring-server`, writes the launchd service `com.localitas.app.homebase-ring` for it and loads it on port 9223.

It starts unconfigured and announces itself to core. Its `health.json` provides `homebase-devices/v1`, so Homebase takes it on as soon as core serves it. Once configured it reads the Ring account's devices every minute (Ring's cloud pushes nothing) and hints Homebase on the `homebase`/`devices` channel when they change; until then its device routes answer 503.

### 3. Configure in Homebase UI

1. Open Homebase (http://localhost:8090/apps/ext/homebase/)
2. Click the **Plugins** button in the sidebar header
3. You should see "Ring" listed as a discovered plugin
4. Paste the Vault `public_id` from step 1 into the credential field
5. Click **Save Credential**

Homebase pushes the credential to homebase-ring, which authenticates with Ring's servers and hints Homebase right away. Your Ring devices appear in Homebase as soon as it reads them.

## Troubleshooting

**"vault secret key not found"** — The Vault credential doesn't have a key named `ring_refresh_token`. Check the credential contents.

**"oauth failed with status 401"** — The refresh token has expired or been revoked. Run `./bin/homebase-ring-server auth` again and update the Vault credential.

**"oauth failed with status 412"** — Ring is requesting 2FA. Run `./bin/homebase-ring-server auth` again and complete the 2FA challenge.

**Devices not appearing** — homebase-ring polls Ring's cloud every minute and, when the devices changed, hints Homebase on cache `homebase`, channel `devices`; Homebase reads the plugin's devices only when hinted. Check that homebase-ring is healthy: `curl http://localhost:9223/health.json`.

## App Store

Install via the Localitas App Store (recommended):

```bash
localitas-core app-store add --name homebase-ring --compose ./docker-compose.yml --port 9223
localitas-core app-store start homebase-ring
```

Or open the App Store UI (package icon, top-right nav) and paste the `docker-compose.yml`.

The image is published to `ghcr.io/localitas/localitas-app-homebase-ring:latest`. To publish a new version:

```bash
make docker-push   # runs tests, builds, and pushes to ghcr.io
```

## License

MIT
