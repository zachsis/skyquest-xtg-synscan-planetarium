package server

// TODO(stretch): LX200 protocol
//
// The LX200 protocol is a text-based, '#'-terminated ASCII protocol used by
// Meade telescopes and supported by many planetarium programs as an alternative
// to the Stellarium binary format.
//
// Planned commands for a future implementation:
//
//   :GR#  → HH:MM:SS#          — get current RA
//   :GD#  → sDD*MM:SS#         — get current Dec
//   :Sr HH:MM:SS#  → "1"       — set target RA
//   :Sd sDD*MM:SS# → "1"       — set target Dec
//   :MS#  → "0"                — slew to target (move & slew)
//   :Q#   → (none)             — cancel slew
//   :GVP# → "OrionTelescope#"  — get product name
//   :GVN# → "1.0#"             — get firmware version
//
// The server would:
//   1. Listen on a separate port (default 4030, configurable).
//   2. Use its own ASCII reader goroutine per client (bufio.Scanner with '#'
//      as the split token — NOT the binary length-prefix reader).
//   3. Maintain targetRA/targetDec state for the Sr/Sd/MS sequence.
//   4. Forward :MS# to GoToService.SlewToCoordinates(targetRA, targetDec).
//   5. Forward :Q# to GoToService.CancelSlew().
//   6. Share the same telescope.PositionProvider as StellariumServer for live
//      RA/Dec responses.
//
// The implementation would closely mirror StellariumServer's lifecycle
// (Start/Stop, context cancellation, SetPositionProvider, SetOnClientChange).
