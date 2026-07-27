# Implementation Notes

This document describes how the emulator is currently structured and records the compatibility choices made in the CPU core.

## Runtime Flow

The executable entry point is `cmd/chip8/main.go`.

At startup:

1. `main` parses an optional `-profile vip|schip` flag (default `vip`) and expects a ROM path argument.
2. `NewGame` maps the profile flag to `chip8.NewVIPProfile()` or `chip8.NewSuperChipProfile()` and creates a `chip8.CPU` with `chip8.NewWithQuirks(quirks)`.
3. The ROM is read from disk and copied into memory starting at `0x200`.
4. Ebiten starts the game loop.

Each Ebiten update:

1. If `cpu.Halted` is set (from opcode `00FD`), `Update` returns `ebiten.Termination` and the loop exits cleanly.
2. `syncInput` clears and rebuilds the CHIP-8 key state from the current keyboard state.
3. The CPU executes `cyclesPerTick` instructions, checking `Halted` between each one.
4. Delay and sound timers are decremented once.

Each draw:

1. The screen is cleared to black.
2. `Layout` and `Draw` read `cpu.DisplayWidth()`/`cpu.DisplayHeight()` each frame, so the window follows the current lo-res/hi-res mode.
3. The CPU display buffer (`cpu.Pixels()`, always strided at `MaxScreenWidth`) is scanned over that logical width/height.
4. Enabled pixels are drawn white.

## CPU State

The CPU state is stored in `internal/chip8/cpu.go`.

Important fields:

- `Memory [4096]byte`: CHIP-8 memory.
- `V [16]byte`: registers `V0` through `VF`.
- `R [8]byte`: SUPER-CHIP RPL flag registers, saved/restored by `FX75`/`FX85`.
- `I uint16`: index register.
- `PC uint16`: program counter.
- `Stack [16]uint16` and `SP byte`: subroutine stack.
- `Hires bool`: current display mode, toggled by `00FE`/`00FF`.
- `Halted bool`: set by `00FD`; once true, `Step()` becomes a no-op.
- `pixels [MaxScreenWidth * MaxScreenHeight]bool` (private): monochrome display buffer, always backed by the full 128x64 grid regardless of the current mode. The exported `Pixels()` method returns a pointer to it for frontend code. The stride is always `MaxScreenWidth` (128) — even in lo-res mode — so index math is `y*MaxScreenWidth + x`, not `y*DisplayWidth() + x`.
- `DelayTimer byte` and `SoundTimer byte`: timers.
- `Keys [16]bool`: CHIP-8 keypad state.

Programs are loaded at `ProgramStart`, currently `0x200`. The built-in font is copied to `FontStart` (`0x050`), and the SUPER-CHIP big font to `BigFontStart` (`0x0A0`).

`DisplayWidth()` and `DisplayHeight()` return the *logical* screen size for the current mode (64x32 or 128x64 depending on `Hires`) — use these for rendering and bounds, not the constants below directly.

The screen-size constants are:

- `LowScreenWidth` / `LowScreenHeight` (64/32): lo-res CHIP-8 display.
- `MaxScreenWidth` / `MaxScreenHeight` (128/64): hi-res SUPER-CHIP display, and the fixed backing-buffer stride.

There is no `ScreenWidth`/`ScreenHeight` pair anymore (an earlier revision of this doc referenced those) — the frontend now reads `DisplayWidth()`/`DisplayHeight()` each frame instead of a single fixed size.

## Fetch, Decode, Execute

`Step` is the public instruction step:

1. `Fetch` reads two bytes at `PC`.
2. `Fetch` advances `PC` by two.
3. `Execute` decodes and applies the opcode.

Because `PC` is advanced during fetch, instructions that repeat the current opcode, such as `FX0A` while waiting for input, subtract two from `PC`.

## Display Behavior

`DXYN` draws an `N` byte sprite at coordinates `(VX, VY)`. When `N` is `0` and `Hires` is `true`, it instead draws a 16x16 sprite (32 bytes, 2 per row) — this is the SUPER-CHIP `DXY0` extension. In lo-res mode, `N == 0` draws nothing, matching classic CHIP-8.

Current behavior:

- Coordinates are reduced modulo the current logical screen dimensions (`DisplayWidth()`/`DisplayHeight()`) before drawing.
- Sprite pixels only wrap horizontally/vertically if `Quirks.WrapSprites` is set; otherwise rows/columns that run off the edge are clipped (`break`/`continue`). Both bundled profiles (`NewVIPProfile()`, `NewSuperChipProfile()`) set `WrapSprites` to `false`, so clipping is the default in practice.
- Pixels are XORed into the display buffer.
- `VF` is set to `1` if any enabled pixel is turned off by collision.
- Sprite memory reads are range checked before drawing.
- The backing pixel buffer is always `MaxScreenWidth x MaxScreenHeight` (128x64); only the logical width/height used for drawing and wrapping changes with `Hires`.

## Scrolling (`00CN`, `00FB`, `00FC`)

SUPER-CHIP adds three scroll opcodes, implemented as `scrollDown`, `scrollRight`, and `scrollLeft` in `cpu.go`. All three operate on the current logical `DisplayWidth()`/`DisplayHeight()`, but index into the buffer with the fixed `MaxScreenWidth` stride.

**Known bugs (not yet fixed — intentionally left as-is for now):**

- `scrollDown` ([cpu.go:676](../internal/chip8/cpu.go#L676)) copies `c.pixels[(y-n)*MaxScreenWidth]` (column 0 of the source row) into every column of the destination row, instead of `c.pixels[(y-n)*MaxScreenWidth+x]`. It collapses each row to a single repeated value rather than shifting per-column data down. Confirmed by drawing a 1-row `0xAA` (`10101010`) sprite and running `00C4`: the destination row comes out fully lit instead of showing the shifted pattern.
- `scrollRight` ([cpu.go:688-697](../internal/chip8/cpu.go#L688-L697)) shifts pixels right by 4 but never blanks the 4 columns it vacates on the left, unlike `scrollLeft`, which does blank its vacated columns. Confirmed by drawing a solid 8-pixel row and running `00FB`: columns `0-3` stay lit with stale data instead of clearing.

## Key Input

The frontend maps the keyboard to CHIP-8 keys:

```text
CHIP-8 keypad       Keyboard
1 2 3 C             1 2 3 4
4 5 6 D             Q W E R
7 8 9 E             A S D F
A 0 B F             Z X C V
```

`EX9E` skips if the key stored in `VX` is currently pressed.

`EXA1` skips if the key stored in `VX` is currently not pressed.

`FX0A` has stateful behavior:

1. If no key is pressed, `PC` is rewound and the instruction waits.
2. Once a key is pressed, the CPU records that key and keeps waiting.
3. After that same key is released, the key value is stored in `VX` and execution continues.

This matches keypad test ROMs that expect `FX0A` to halt until release.

## Compatibility Choices

CHIP-8 has several historical interpreter variants. Compatibility differences are now modeled as a `Quirks` struct (`internal/chip8/cpu.go`), with two named presets:

| `Quirks` field | Opcodes affected | `NewVIPProfile()` (`-profile vip`) | `NewSuperChipProfile()` (`-profile schip`) |
| --- | --- | --- | --- |
| `LogicResetsVF` | `8XY1`, `8XY2`, `8XY3` | `true` (resets `VF` to `0`) | `false` (leaves `VF` alone) |
| `ShiftModifiesVY` | `8XY6`, `8XYE` | `true` (shifts `VY`, stores in `VX`) | `false` (shifts `VX` in place) |
| `StoreLoadMutatesI` | `FX55`, `FX65` | `true` (`I` advances by count) | `false` (`I` unchanged) |
| `JumpUsesVX` | `BNNN`/`BXNN` | `false` (jumps to `NNN + V0`) | `true` (jumps to `NNN + VX`, `X` from the opcode) |
| `IndexOverFlowsVF` | `FX1E` | `false` (`VF` untouched) | `false` (`VF` untouched) |
| `WrapSprites` | `DXYN`/`DXY0` | `false` (sprites clip at edges) | `false` (sprites clip at edges) |

Two ways to get a `CPU`:

- `chip8.New()` — no profile argument, defaults to `NewSuperChipProfile()`.
- `chip8.NewWithQuirks(quirks)` — explicit profile; this is what `cmd/chip8/main.go` uses, defaulting to `NewVIPProfile()` via its own `-profile` flag default.

These two defaults disagree (library default is SCHIP, CLI default is VIP). `internal/chip8/cpu_test.go` has several tests (`Test8XY1Or`, `Test8XY2And`, `Test8XY3Xor`, `Test8XY6ShiftRight`, `Test8XYEShiftLeft`, `TestBNNNJumpToV0PlusNNN`) that call `chip8.New()` and assert VIP-profile behavior, so they currently fail against the SCHIP default — see the Testing Strategy section below.

Also note: `FX0A` still waits for key release before continuing in both profiles (this isn't currently a `Quirks` field).

## Implemented Opcode Groups

The CPU currently implements:

- `00CN` (scroll down), `00E0`, `00EE`
- `00FB` (scroll right), `00FC` (scroll left), `00FD` (halt), `00FE`/`00FF` (lo-res/hi-res toggle)
- `1NNN`, `2NNN`
- `3XNN`, `4XNN`, `5XY0`, `9XY0`
- `6XNN`, `7XNN`
- `8XY0`, `8XY1`, `8XY2`, `8XY3`, `8XY4`, `8XY5`, `8XY6`, `8XY7`, `8XYE`
- `ANNN`, `BNNN`/`BXNN`, `CXNN`, `DXYN` (and `DXY0` 16x16 sprites in hi-res mode)
- `EX9E`, `EXA1`
- `FX07`, `FX0A`, `FX15`, `FX18`, `FX1E`, `FX29`, `FX30` (big-font address), `FX33`, `FX55`, `FX65`, `FX75`/`FX85` (RPL flags)

Unknown opcodes return an error from `Execute`, which bubbles up through `Step` and stops the frontend loop. Once `00FD` sets `Halted`, `Step()` short-circuits to a no-op instead of erroring.

## Testing Strategy

Unit tests live in `internal/chip8/cpu_test.go`.

The tests execute small in-memory programs or call `Execute` directly. They cover:

- Program loading and fetch/step behavior
- Jumps, calls, returns, and skips
- Arithmetic and flag behavior
- Display drawing, wrapping, and collision
- Key skip and key wait behavior
- Timer reads/writes and timer decrementing
- BCD storage and register memory operations

Run tests with:

```powershell
go test ./...
```

**Six tests currently fail**: `Test8XY1Or`, `Test8XY2And`, `Test8XY3Xor`, `Test8XY6ShiftRight`, `Test8XYEShiftLeft`, `TestBNNNJumpToV0PlusNNN`. All six call `New()` and assert VIP-profile quirk behavior, but `New()` now defaults to `NewSuperChipProfile()` (see Compatibility Choices above), so the asserted defaults no longer hold. This is a genuine default-vs-test mismatch to resolve — either change what `New()` defaults to, or point these tests at `NewWithQuirks(NewVIPProfile())` explicitly (the neighboring `_ModernProfile` tests already show that pattern for the opposite case).

There is currently no unit test coverage in `cpu_test.go` for any of the SUPER-CHIP-specific opcodes (`00CN`, `00FB`, `00FC`, `00FD`, `00FE`/`00FF`, `DXY0`, `FX75`/`FX85`). A validation pass instead exercised them with small throwaway hand-built programs run directly against the `CPU` (load bytes with `LoadProgram`, `Step()` N times, inspect `Pixels()`/`V`/`R`/`Hires`/`Halted`), which is how the two scrolling bugs above were found. That approach isn't committed anywhere — turning it into permanent `cpu_test.go` cases (one assertion per opcode, following the existing test style) is still open work.

Manual ROM checks can be run from the repository root:

```powershell
go run ./cmd/chip8 ./roms/chip8-logo.ch8
go run ./cmd/chip8 ./roms/ibm.ch8
go run ./cmd/chip8 ./roms/corax.ch8
go run ./cmd/chip8 ./roms/flags.ch8
go run ./cmd/chip8 -profile schip ./roms/quirks.ch8
```

`quirks.ch8` (Timendus's `chip8-test-suite` quirks test) has an interactive platform-select menu — press `3` on the mapped keyboard to test the SCHIP profile — followed by a visual pass/fail readout per quirk. It was run headlessly far enough to confirm the menu renders, but reading the actual quirk results still requires playing it in the Ebiten window; there's no framebuffer assertion harness to automate that yet.

## Good Next Improvements

- Fix `scrollDown`'s missing `+x` and `scrollRight`'s missing vacated-column clear (see Scrolling section above).
- Resolve the `New()` vs. CLI default-profile mismatch, and update or rewrite the six tests that currently fail because of it.
- Add unit test coverage in `cpu_test.go` for `00CN`, `00FB`, `00FC`, `00FD`, `00FE`/`00FF`, `DXY0`, and `FX75`/`FX85`.
- Drive delay and sound timers from a clear 60 Hz timing model.
- Add simple audio output for `SoundTimer`.
- Add automated screenshot or framebuffer assertions for known test ROMs (would let `quirks.ch8`'s SCHIP results be checked without playing it manually).
- Split CPU compatibility settings from frontend concerns.
