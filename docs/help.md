# Homebase Ring Plugin

Ring device integration for Homebase.

## Supported Devices

- **Doorbells** — all Ring Video Doorbell models (v2-v5, Pro, Pro 2, Elite, Wired)
- **Cameras** — Stick Up Cam, Spotlight Cam, Floodlight Cam (all variants)
- **Chimes** — Ring Chime, Chime Pro
- **Base Stations** — Ring Alarm base station

## Commands

| Command | Description | Arguments |
|---------|-------------|-----------|
| `light_on` | Turn on floodlight/spotlight | — |
| `light_off` | Turn off floodlight/spotlight | — |
| `siren_on` | Activate siren | — |
| `siren_off` | Deactivate siren | — |
| `chime_play` | Play a sound on chime | `{"kind": "ding"}` or `{"kind": "motion"}` |
| `chime_volume` | Set chime volume (0-11) | `{"volume": 5}` |

## Permissions

| Scope | Allowed |
|-------|---------|
| read | List devices, get device state, view snapshots, health check |
| write | Send commands to devices |
| admin | Configure plugin credentials |

## API Endpoints

- `GET /api/devices` — list all Ring devices (read)
- `GET /api/devices/{id}` — get device details (read)
- `GET /api/devices/{id}/health` — device health/connectivity (read)
- `GET /api/devices/{id}/snapshot` — camera snapshot, JPEG (read)
- `POST /api/devices/{id}/command` — send command (write)
- `POST /api/configure` — set vault credential (admin, called by Homebase)
