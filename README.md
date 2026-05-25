# OrionTelescope

A cross-platform desktop application for controlling SynScan-compatible telescope mounts with an integrated planetarium, object browser, and observation journal.

Built with Go and [Fyne](https://fyne.io/) — no CGO required, produces a single static binary for macOS, Linux, and Windows.

## Features

- **Live telescope control** — connect to SynScan mounts over serial, with real-time position display, GoTo slewing, tracking modes, and alignment wizard
- **Interactive sky chart** — stereographic all-sky projection with 15,000+ stars (HYG 4.2), constellation lines and labels, telescope crosshair with FOV circle, and click-to-select
- **Deep-sky object overlays** — galaxies, nebulae, clusters, and planets rendered with distinct symbols on the sky chart
- **Solar system ephemeris** — real-time Sun, Moon, and planet positions computed via VSOP87B planetary theory, with phase and elongation data
- **Object browser** — search and filter 10,000+ objects (Messier, NGC/IC, named stars, planets) by type, catalog, magnitude, and visibility; sort by altitude to find tonight's best targets
- **Observation journal** — auto-logs every GoTo command with timestamp and target; browse, edit, rate seeing/transparency, add equipment notes, and export sessions to CSV
- **Stellarium integration** — built-in TCP server speaks the Stellarium Telescope Protocol so planetarium software can read telescope position and send GoTo commands

## Requirements

- Go 1.22+ (the project uses Go 1.26 but should build with 1.22+)
- A SynScan-compatible telescope mount and serial/USB adapter (for telescope control features; the planetarium and catalog features work standalone)

### Platform dependencies

Fyne requires platform graphics libraries:

- **macOS** — Xcode command line tools (`xcode-select --install`)
- **Linux** — `gcc`, `libgl1-mesa-dev`, `xorg-dev` (Debian/Ubuntu) or equivalent
- **Windows** — a C compiler (e.g., TDM-GCC) for Fyne's OpenGL bindings

## Quick start

```bash
# Clone
git clone https://github.com/zachsis/skyquest-xtg-synscan-planetarium.git
cd skyquest-xtg-synscan-planetarium

# Build and run
make run

# Or build separately
make build
./bin/oriontelescope
```

On first launch the app creates `~/.oriontelescope/` with a default config file and an empty observation database.

## Configuration

Open the **Settings** panel (last item in the sidebar) to configure:

| Setting | Description | Default |
|---------|-------------|---------|
| Location Name | Label for your observing site | — |
| Latitude | Observer latitude in decimal degrees | 0.0 |
| Longitude | Observer longitude in decimal degrees | 0.0 |
| Elevation | Observer elevation in meters | 0.0 |
| Serial Port | Path to the SynScan serial device (e.g., `/dev/tty.usbserial-1420`) | — |
| Baud Rate | Serial baud rate | 9600 |
| Magnitude Cutoff | Faintest magnitude for "Tonight's Best" filter | 10.0 |

Settings are saved to `~/.oriontelescope/config.json` (or `$XDG_CONFIG_HOME/oriontelescope/config.json` on Linux).

## Usage

### Without a telescope

The sky chart, object browser, and ephemeris all work without a telescope connection. Set your observer location in Settings to get accurate rise/set times, altitude calculations, and a correctly oriented sky chart.

### Connecting a telescope

1. Plug in your SynScan hand controller via USB-serial adapter
2. In **Settings**, select the serial port and save
3. The **Status** panel shows connection state and live RA/Dec/Alt/Az once connected
4. Use **GoTo** to slew to coordinates or named targets
5. Use **Tracking** to change tracking modes (sidereal, lunar, solar, or stop)
6. Use **Alignment** for star alignment procedures

### Object Browser

The **Objects** panel lets you search the full catalog. Use filters to narrow by type (galaxy, nebula, cluster, star, planet), catalog source (Messier, NGC, named stars), magnitude, or above-horizon visibility. Click **Tonight's Best** for a pre-filtered list of the brightest objects currently above the horizon, sorted by altitude.

Click any object for details, then **GoTo** to slew the telescope or **Show on Chart** to center the sky chart on it.

### Observation Journal

Every GoTo slew is automatically logged in the **Log** panel with timestamp, target, and coordinates. You can:

- Browse past sessions and their observations
- Edit entries to add notes, seeing/transparency ratings (1-5), and equipment details
- Add manual entries with catalog autocomplete
- Export sessions to CSV

The journal is stored in `~/.oriontelescope/observations.db` (SQLite).

### Stellarium Integration

To control the telescope from Stellarium or other planetarium software:

1. In **Settings**, scroll to the Stellarium Server section
2. Set the port (default 10001) and enable the server
3. In Stellarium: Configuration > Plugins > Telescope Control > Configure > Add
4. Choose "External software or a remote computer", set host to `localhost`, port to `10001`
5. Connect — the telescope crosshair appears in Stellarium and tracks your mount's position
6. Use Stellarium's "Slew to" to send GoTo commands

> On macOS, the OS may prompt you to allow incoming network connections the first time the server starts. This is expected.

## Project structure

```
cmd/oriontelescope/       Application entry point
internal/
  astro/                  Coordinate transforms, rise/set/transit calculations
  catalog/                Star catalog (HYG 4.2), Messier/NGC/named star data, registry
  config/                 Observer settings, persistence
  ephemeris/              Solar system positions (VSOP87B), planet info
  logging/                Observation journal, SQLite storage, CSV export
  serial/                 Cross-platform serial port layer
  server/                 Stellarium telescope protocol TCP server
  slew/                   GoTo service with mutual exclusion and events
  synscan/                SynScan hand controller protocol driver
  telescope/              Position provider interface, state publishing
  ui/                     Fyne UI panels (status, goto, tracking, alignment,
                          sky chart, objects, log, settings)
  ui/skychart/            Sky chart canvas, stereographic projection, overlays
  ui/panels/              Settings panel
```

## Development

```bash
make build      # Compile to bin/oriontelescope
make run        # Build and run
make test       # Run all tests
make lint       # go vet + staticcheck
make clean      # Remove build artifacts
```

### Regenerating embedded data

The star catalog, NGC data, and VSOP87B ephemeris files are committed to the repo. To regenerate from upstream sources:

```bash
make generate-catalog   # Download HYG 4.2, filter to mag <= 7.0
make generate-ngc       # Download OpenNGC, filter to objects with coordinates
make generate-vsop87    # Download VSOP87B planetary theory data files
```

## Third-party data

See [THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md) for attribution of embedded astronomical data (HYG star catalog, OpenNGC, VSOP87B).
