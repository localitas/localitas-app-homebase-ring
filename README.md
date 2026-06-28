# Homebase Ring Plugin

Ring device integration for Homebase. Controls Ring doorbells, cameras, chimes, and alarm systems.

## Acquiring a Ring Refresh Token

Ring uses OAuth2 with refresh tokens. Generate one using the built-in `auth` subcommand:

```bash
cd apps/homebase-ring && make build
./bin/homebase-ring-server auth
```

This prompts for your Ring email and password. If you have 2FA enabled (recommended), it will ask for the code sent to your phone or authenticator app. On success, it prints the refresh token and a ready-to-use `curl` command to store it in Vault.

The refresh token rotates automatically. Homebase-ring writes updated tokens back to Vault, so you only need to do this once.

## Setup

### 1. Store the Token in Vault

After running `auth`, use the printed `curl` command, or manually:

```bash
curl -X POST http://localhost:8080/apps/vault/api/credentials \
  -H "Authorization: Bearer $(cat ~/.localitas/api-token)" \
  -H "Content-Type: application/json" \
  -d '{"name": "Ring", "data": {"ring_refresh_token": "YOUR_REFRESH_TOKEN_HERE"}}'
```

Note the `public_id` in the response — you'll need it in step 3.

### 2. Start homebase-ring

```bash
cd apps/homebase-ring && make start
```

It starts unconfigured and broadcasts via mDNS. Homebase discovers it automatically.

### 3. Configure in Homebase UI

1. Open Homebase (http://localhost:8080/apps/ext/homebase/)
2. Click the **Plugins** button in the sidebar header
3. You should see "Ring" listed as a discovered plugin
4. Paste the Vault `public_id` from step 1 into the credential field
5. Click **Save Credential**

Homebase pushes the credential to homebase-ring, which authenticates with Ring's servers. Your Ring devices will appear in Homebase within 60 seconds.

## Troubleshooting

**"vault secret key not found"** — The Vault credential doesn't have a key named `ring_refresh_token`. Check the credential contents.

**"oauth failed with status 401"** — The refresh token has expired or been revoked. Run `./bin/homebase-ring-server auth` again and update the Vault credential.

**"oauth failed with status 412"** — Ring is requesting 2FA. Run `./bin/homebase-ring-server auth` again and complete the 2FA challenge.

**Devices not appearing** — Homebase syncs plugin devices every 60 seconds. Check that homebase-ring is healthy: `curl http://localhost:9223/health.json`.

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
