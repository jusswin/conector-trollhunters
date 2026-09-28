# Conector Trollhunters

A one-click Windows launcher that opens a **local command bridge** and exposes it
to the internet through a **Cloudflare quick tunnel** — then copies the public
URL to your clipboard, ready to paste into a chat.

Built for remote-assistance workflows: an AI assistant (or a helper) can send
commands to your PC through the bridge while you keep both windows open. If the
tunnel drops, it **reconnects automatically** and copies the new URL.

## How it works

1. Starts a PowerShell `HttpListener` bridge on `http://localhost:8080`
   (embedded `puente_embed.ps1`). Every request must carry the `X-Token` header
   matching your secret token, otherwise it gets a `403`.
2. Downloads the official `cloudflared` binary on first run (only once).
3. Opens a Cloudflare quick tunnel (`https://….trycloudflare.com`) to the bridge.
4. Copies the tunnel URL to the clipboard and prints it.
5. Watches the tunnel forever: on failure it waits 5 s, kills stale
   `cloudflared` processes, revives the bridge if needed, and reopens the tunnel.

## Requirements

- Windows 10/11 (x86-64)
- Internet access (to download `cloudflared` once)

## Usage

1. Open `puente_embed.ps1` and set `$token` to a long random secret.
   **The same token must be configured on the assistant side** — it is sent as
   the `X-Token` header with every request.
2. Double-click `ConectorTrollhunters.exe`.
3. Paste the shown URL wherever you need it.
4. Keep both windows (bridge + connector) open while working.
   To disconnect, just close them.

## Build from source

```powershell
# requires Go 1.21+
go build -o ConectorTrollhunters.exe .
```

Run the tests:

```powershell
go test ./...
```

## Security notes

- The bridge executes arbitrary commands sent with a valid token. **Use a strong
  token, keep it private, and only share the tunnel URL with someone you trust.**
- The tunnel URL is public to whoever knows it; the token is the only gate.
- Nothing is installed: `cloudflared.exe` is downloaded next to the program and
  the bridge script is written to `%LOCALAPPDATA%\Trollhunters`.

## License

MIT — see [LICENSE](LICENSE).
