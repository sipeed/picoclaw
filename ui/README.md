# PicoClaw Web UI

This folder contains a small browser-based chat UI for PicoClaw.

The gateway process serves the built assets and exposes a WebSocket endpoint for chat.

## Build

From the repository root:

```bash
make ui-build
```

Or directly:

```bash
cd ui
npm install
npm run build
```

The production build output is written to `ui/dist/`.

## Run

Start the gateway:

```bash
picoclaw gateway
```

Then open in your browser:

- `http://<gateway-host>:18790/`

## Authentication (shared token)

If `gateway.token` is set, you must provide it as a query param when opening the UI:

- `http://<gateway-host>:18790/?token=<secret>`

The UI reads the `token` from the page URL and passes it to the WebSocket connection.

## Bind mode

The gateway listen address is controlled by `gateway.bind`:

- `local`: binds to `127.0.0.1`
- `tailnet`: binds to the host's Tailscale address (IPv4 in `100.64.0.0/10` or `10.0.0.0/8`)
- `all`: binds to `0.0.0.0`

See the main config example (`config/config.example.json`) for configuration keys.
