package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"

	"github.com/JustTryingToDoBetter/go-chip8/internal/chip8"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	windowScale   = 12
	cyclesPerTick = 10
)

type Game struct {
	cpu *chip8.CPU
}

func NewGame(romPath string, profile string) (*Game, error) {
	// Select profile quirks
	var quirks chip8.Quirks
	switch profile {
	case "schip", "superchip":
		quirks = chip8.NewSuperChipProfile()
	case "vip", "chip8", "default":
		quirks = chip8.NewVIPProfile()
	default:
		return nil, fmt.Errorf("unknown profile: %q (valid options: vip, schip)", profile)
	}

	cpu := chip8.NewWithQuirks(quirks)

	if err := cpu.LoadROM(romPath); err != nil {
		return nil, err
	}

	return &Game{
		cpu: cpu,
	}, nil
}

func (g *Game) Update() error {
	// Clean exit path when CPU encounters a halt signal
	if g.cpu.Halted {
		return ebiten.Termination
	}

	g.syncInput()

	for i := 0; i < cyclesPerTick; i++ {
		if g.cpu.Halted {
			return ebiten.Termination
		}

		if err := g.cpu.Step(); err != nil {
			return err
		}
	}

	g.cpu.UpdateTimers()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	width := g.cpu.DisplayWidth()
	height := g.cpu.DisplayHeight()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// CRITICAL: Stride is always MaxScreenWidth (128) for the 128x64 backing array
			index := y*chip8.MaxScreenWidth + x

			if g.cpu.Pixels()[index] { // Or g.cpu.PixelAt(x, y) if you export a getter
				screen.Set(x, y, color.White)
			}
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Dynamically report logical internal resolution to Ebitengine
	return g.cpu.DisplayWidth(), g.cpu.DisplayHeight()
}

func (g *Game) syncInput() {
	for i := range g.cpu.Keys {
		g.cpu.Keys[i] = false
	}

	keyMap := map[ebiten.Key]byte{
		ebiten.KeyX: 0x0,
		ebiten.Key1: 0x1,
		ebiten.Key2: 0x2,
		ebiten.Key3: 0x3,
		ebiten.KeyQ: 0x4,
		ebiten.KeyW: 0x5,
		ebiten.KeyE: 0x6,
		ebiten.KeyA: 0x7,
		ebiten.KeyS: 0x8,
		ebiten.KeyD: 0x9,
		ebiten.KeyZ: 0xA,
		ebiten.KeyC: 0xB,
		ebiten.Key4: 0xC,
		ebiten.KeyR: 0xD,
		ebiten.KeyF: 0xE,
		ebiten.KeyV: 0xF,
	}

	for physicalKey, chip8Key := range keyMap {
		if ebiten.IsKeyPressed(physicalKey) {
			g.cpu.Keys[chip8Key] = true
		}
	}
}

func main() {
	profileFlag := flag.String("profile", "vip", "hardware profile quirks (vip, schip)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("usage: chip8 [-profile vip|schip] <rom-path>")
	}

	romPath := args[0]

	game, err := NewGame(romPath, *profileFlag)
	if err != nil {
		log.Fatal(err)
	}

	// Base window dimensioned to low-res standard scale
	ebiten.SetWindowSize(
		chip8.LowScreenWidth*windowScale,
		chip8.LowScreenHeight*windowScale,
	)
	ebiten.SetWindowTitle("go-chip8")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
