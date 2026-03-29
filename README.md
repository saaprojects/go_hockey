# Go Hockey

## Current layout

- `cmd/hockey-v2`: executable entrypoint
- `internal/client`: solo and online Ebitengine clients
- `internal/sim`: gameplay state and fixed-tick simulation core
- `internal/netcode`: network message shapes and TCP client
- `internal/server`: authoritative multiplayer match server

## Launcher

```powershell
go run ./cmd/hockey-v2
```

This opens the same Ebitengine client launcher for every mode.

From that menu you can choose:

- `Solo Game`
- `Host Multiplayer`
- `Join Multiplayer`

Launcher controls:

- `Up` and `Down` or `W` and `S`: move between menu options
- `Enter` or `Space`: launch the selected option
- When `Join Multiplayer` is selected, type to edit the server address
- `Backspace`: delete one character from the join address

## In-match controls

- `W`, `A`, `S`, `D` or arrow keys: move
- `Shift`: pass
- `Space`: shoot or poke-check
- `Tab`: switch to the skater closest to the puck
- `P`: pause
- `R`: restart
- `Esc`: return to the launcher from solo or multiplayer

## CLI shortcuts

The menu is the default entry point, but the direct commands still work if you want to jump straight into a mode.

## Local multiplayer try-it-now

Start a local server and immediately join it with one client:

```powershell
go run ./cmd/hockey-v2 -host
```

Then open a second terminal for the other player:

```powershell
go run ./cmd/hockey-v2 -join <your local IP>:4242
```

The first player to join gets `home`. The second gets `away`.

## Online lobby and intermission controls

During the pregame ready screen:

- `A` or `Left Arrow`: previous color
- `D` or `Right Arrow`: next color
- `Space` or `Enter`: toggle ready
- The match will not begin until both players are ready

During intermission:

- The same ready and color controls apply
- The game auto-continues after 10 seconds if nobody changes anything
- The overlay shows the completed period's shots on goal and goals for both teams

Available team colors:

- Black
- Orange
- Green
- Blue
- Red

The two teams cannot lock in the same color.

## Dedicated server

Run only the authoritative match server:

```powershell
go run ./cmd/hockey-v2 -server -listen :4242
```

Join from another machine or terminal:

```powershell
go run ./cmd/hockey-v2 -join YOUR_HOST_OR_IP:4242
```

## Online client controls

During play:

- `W`, `A`, `S`, `D` or arrow keys: move
- `Shift`: pass
- `Space`: shoot or poke-check
- `Tab`: switch to the skater closest to the puck

Notes:

- The current networking path is server-authoritative.
- This first slice sends full snapshots over TCP, so it is a real multiplayer prototype, not the final low-latency netcode.
- Solo mode still runs entirely local.
- Multiplayer now includes pregame color selection, intermission ready-up screens, and simple period stats.

## Headless smoke test

```powershell
go run ./cmd/hockey-v2 -headless
```

## Rewrite direction

1. Keep the sim authoritative and fixed-tick.
2. Use the same sim for solo, bot matches, and online play.
3. Run multiplayer through a dedicated server process or `-host` mode.
4. Replace the naive full-snapshot TCP path with prediction/interpolation once the baseline online loop feels good.
