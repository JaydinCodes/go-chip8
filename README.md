# go-chip8

A CHIP-8 emulator written in Go. The emulator core lives in `internal/chip8`, and the runnable desktop frontend uses Ebiten from `cmd/chip8`.

## Features

- 4 KiB CHIP-8 memory with programs loaded at `0x200`
- Built-in CHIP-8 font (`0x050`) and SUPER-CHIP big font (`0x0A0`)
- 64x32 lo-res and 128x64 hi-res (SUPER-CHIP) monochrome display, selected with `00FE`/`00FF`
- 16 8-bit general registers `V0`-`VF`, plus 8 RPL flag registers (`FX75`/`FX85`)
- Index register `I`, program counter `PC`, stack, delay timer, and sound timer
- CHIP-8 keypad state with the common keyboard mapping
- Selectable compatibility profiles (`-profile vip` or `-profile schip`) covering the classic CHIP-8/VIP quirks and the SUPER-CHIP quirks
- Scroll opcodes (`00CN`, `00FB`, `00FC`) and the `00FD` halt opcode
- Unit tests for core opcodes, drawing, timers, register storage, and key waits
- Bundled test ROMs in `roms/`

## Project Structure

```text
go-chip8/
├─ cmd/
│  └─ chip8/
│     └─ main.go          # Ebiten window, input mapping, emulator loop
├─ internal/
│  └─ chip8/
│     ├─ cpu.go           # CPU state, opcode fetch/decode/execute, display, timers
│     ├─ cpu_test.go      # Unit tests for implemented CHIP-8 behavior
│     └─ font.go          # CHIP-8 font sprites
├─ roms/                  # Local test ROMs
├─ go.mod
└─ go.sum
```

More implementation detail is in [docs/implementation.md](docs/implementation.md).

## Requirements

- Go installed
- A desktop environment supported by Ebiten

The project currently uses Ebiten for windowing, rendering, and keyboard input.

## Running

Run the emulator with a ROM path:

```powershell
go run ./cmd/chip8 ./roms/ibm.ch8
```

Pick a compatibility profile with `-profile` (defaults to `vip`):

```powershell
go run ./cmd/chip8 -profile schip ./roms/quirks.ch8
```

Other useful bundled ROMs:

```powershell
go run ./cmd/chip8 ./roms/chip8-logo.ch8
go run ./cmd/chip8 ./roms/corax.ch8
go run ./cmd/chip8 ./roms/flags.ch8
go run ./cmd/chip8 -profile schip ./roms/quirks.ch8
```

`quirks.ch8` is Timendus's interactive `chip8-test-suite` quirks test: it shows a platform-select menu (press `3` on the CHIP-8 keypad, mapped to keyboard `3`, to test SUPER-CHIP quirks) before running the actual quirk checks and rendering pass/fail on screen. It has to be read visually in the running window — see [Known Limitations](#known-limitations) for why this can't be checked headlessly yet.

The binary expects a ROM path, with an optional `-profile` flag:

```text
chip8 [-profile vip|schip] <rom-path>
```

## Controls

The CHIP-8 keypad is mapped to the keyboard like this:

```text
CHIP-8 keypad       Keyboard
1 2 3 C             1 2 3 4
4 5 6 D             Q W E R
7 8 9 E             A S D F
A 0 B F             Z X C V
```

## Testing

Run all tests:

```powershell
go test ./...
```

Run only the CPU package tests:

```powershell
go test ./internal/chip8
```

The unit tests cover opcode execution at the CPU level. The ROMs in `roms/` are useful for manual compatibility checks in the Ebiten frontend.

**Known failing tests:** six tests (`Test8XY1Or`, `Test8XY2And`, `Test8XY3Xor`, `Test8XY6ShiftRight`, `Test8XYEShiftLeft`, `TestBNNNJumpToV0PlusNNN`) currently fail. They assert classic/VIP quirk behavior (`LogicResetsVF`, `ShiftModifiesVY`, `JumpUsesVX = false`) but call `New()`, which now defaults to `NewSuperChipProfile()` instead of a VIP-like baseline. This is a real default-vs-expectation mismatch introduced when SUPER-CHIP support was added, not a compile issue — see `internal/chip8/cpu_test.go` around lines 242, 268, 294, 514, 587, and 676. It needs a decision: either change `New()`'s default back to a VIP profile, or update these tests to construct the profile they mean to exercise (mirroring the `_ModernProfile` tests next to each one).

### SCHIP opcode validation

There's no automated framebuffer/ROM assertion harness yet (see Known Limitations), and the GUI frontend can't be inspected headlessly. For this pass, SCHIP-specific opcodes were validated with small hand-built programs run against the CPU directly (draw sprite, dump the pixel buffer, compare to the expected pattern):

| Opcode(s) | Behavior checked | Result |
| --- | --- | --- |
| `00FF`/`00FE`, `DXY0` | Hi-res toggle + 16x16 sprite draw, column-to-byte mapping | Correct |
| `FX75`/`FX85` | Save/restore `V0`-`VX` to/from RPL flags `R0`-`R7` | Correct |
| `00FD` | Halt stops further `Step()` execution | Correct |
| `00FC` (scroll left) | Vacated columns cleared after shift | Correct |
| `00CN` (scroll down) | Shift rows down by `N`, preserving per-column pixel values | **Bug** — see below |
| `00FB` (scroll right) | Vacated columns cleared after shift | **Bug** — see below |

Two real bugs were found and are left for you to fix (per the "don't code SUPER-CHIP features for me" approach):

- **`scrollDown` drops per-column data** ([internal/chip8/cpu.go:676](internal/chip8/cpu.go#L676)): `c.pixels[y*MaxScreenWidth+x] = c.pixels[(y-n)*MaxScreenWidth]` is missing `+x` on the right-hand side, so every column in the destination row is set to the value of *column 0* of the source row, instead of shifting each column independently. Repro: draw a 1-row sprite with byte `0xAA` (`10101010`) at `(0,0)`, then execute `00C4` — the whole destination row lights up instead of showing the shifted `10101010` pattern.
- **`scrollRight` doesn't clear vacated columns** ([internal/chip8/cpu.go:688-697](internal/chip8/cpu.go#L688-L697)): unlike `scrollLeft`, which explicitly blanks the columns it vacates, `scrollRight` shifts pixels right by 4 but never blanks columns `0..3` afterward, so old pixel data lingers there. Repro: draw a solid 8-pixel-wide row at `(0,0)`, then execute `00FB` — columns `0-3` stay lit instead of clearing.

`quirks.ch8` itself was run headlessly far enough to render its platform-select menu, but driving it through to the actual SCHIP quirks results screen and reading pass/fail requires playing it in the real GUI window — that part wasn't reliably reproducible from an ASCII dump in this pass.

## Current CHIP-8 Behavior

The core implements two named compatibility profiles via the `Quirks` struct, selected on the CLI with `-profile`:

- `NewVIPProfile()` (CLI: `-profile vip`, the CLI default): `LogicResetsVF`, `ShiftModifiesVY`, and `StoreLoadMutatesI` are all `true`; `JumpUsesVX` and `IndexOverFlowsVF` are `false`.
- `NewSuperChipProfile()` (CLI: `-profile schip`): the inverse — `JumpUsesVX` is `true`, the rest are `false`.

Note the asymmetry: the CLI (`cmd/chip8/main.go`) defaults to the VIP profile, but the library-level `chip8.New()` constructor defaults to the SUPER-CHIP profile (`NewWithQuirks(NewSuperChipProfile())`). Keep that in mind if you construct a `CPU` directly instead of going through the CLI — see the failing-tests note above for where this already caused a mismatch.

Other notable behavior:

- `FX55` and `FX65` do not increment `I` unless `StoreLoadMutatesI` is set.
- `FX0A` waits for a key press and then waits for that key to be released before storing it in `VX`.
- Sprites do not wrap around screen edges by default (`WrapSprites` is `false` in both profiles); wrapping is opt-in per test via `cpu.Quirks.WrapSprites = true`.
- `00FE`/`00FF` switch between 64x32 and 128x64 display and clear the screen.
- `DXY0` draws a 16x16 sprite instead of an 8xN sprite when the display is in hi-res mode.
- `00CN`/`00FB`/`00FC` scroll the display down/right/left (see the bugs noted above for the latter two).
- `00FD` halts the CPU; `Step()` becomes a no-op afterward and the frontend exits cleanly.

## Known Limitations

- No audio output is wired to the sound timer yet.
- Timers are decremented once per Ebiten update, not from a separate 60 Hz clock abstraction.
- Rendering uses direct per-pixel `Set` calls, which is simple but not optimized.
- The emulator exits when it encounters an unknown opcode.
- No unit test coverage yet for the SCHIP-specific opcodes (`00CN`, `00FB`, `00FC`, `00FE`, `00FF`, `DXY0`'s 16x16 mode, `FX75`/`FX85`) — this pass validated them with throwaway hand-built programs instead of adding them to `cpu_test.go`.
- No automated screenshot or framebuffer assertions for ROM-driven test suites like `quirks.ch8`; verifying those still means running the Ebiten window and reading the screen.

## Development Notes

When changing CPU behavior, prefer adding a small opcode-level test in `internal/chip8/cpu_test.go`. This keeps compatibility decisions explicit and makes regressions easier to diagnose before testing full ROMs.
