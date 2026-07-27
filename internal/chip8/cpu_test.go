// internal/chip8/cpu_test.go
package chip8

import "testing"

func stepN(t *testing.T, cpu *CPU, n int) {
	t.Helper()

	for i := 0; i < n; i++ {
		if err := cpu.Step(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadProgram(t *testing.T) {
	cpu := New()

	program := []byte{0x60, 0x0A, 0x61, 0x05}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	if cpu.Memory[0x200] != 0x60 {
		t.Fatalf("expected memory[0x200] to be 0x60, got 0x%02X", cpu.Memory[0x200])
	}

	if cpu.Memory[0x201] != 0x0A {
		t.Fatalf("expected memory[0x201] to be 0x0A, got 0x%02X", cpu.Memory[0x201])
	}

	if cpu.PC != 0x200 {
		t.Fatalf("expected PC to start at 0x200, got 0x%03X", cpu.PC)
	}
}

func TestSetRegisterAndAdd(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0x0A, // V0 = 10
		0x70, 0x01, // V0 += 1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 2)

	if cpu.V[0] != 11 {
		t.Fatalf("expected V0 to be 11, got %d", cpu.V[0])
	}
}

func TestJump(t *testing.T) {
	cpu := New()

	program := []byte{
		0x12, 0x06, // jump to 0x206
		0x60, 0xFF, // skipped
		0x60, 0xEE, // skipped
		0x61, 0x0A, // V1 = 10
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 1)

	if cpu.PC != 0x206 {
		t.Fatalf("expected PC to be 0x206, got 0x%03X", cpu.PC)
	}

	stepN(t, cpu, 1)

	if cpu.V[1] != 10 {
		t.Fatalf("expected V1 to be 10, got %d", cpu.V[1])
	}
}

func TestCallAndReturn(t *testing.T) {
	cpu := New()

	program := []byte{
		0x22, 0x06, // 0x200: call 0x206
		0x60, 0x01, // 0x202: V0 = 1
		0x00, 0x00, // 0x204: unused
		0x61, 0x02, // 0x206: V1 = 2
		0x00, 0xEE, // 0x208: return
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 1)

	if cpu.PC != 0x206 {
		t.Fatalf("expected PC to be 0x206 after CALL, got 0x%03X", cpu.PC)
	}

	if cpu.SP != 1 {
		t.Fatalf("expected SP to be 1 after CALL, got %d", cpu.SP)
	}

	if cpu.Stack[0] != 0x202 {
		t.Fatalf("expected return address 0x202, got 0x%03X", cpu.Stack[0])
	}

	stepN(t, cpu, 1)

	if cpu.V[1] != 2 {
		t.Fatalf("expected V1 to be 2, got %d", cpu.V[1])
	}

	stepN(t, cpu, 1)

	if cpu.PC != 0x202 {
		t.Fatalf("expected PC to return to 0x202, got 0x%03X", cpu.PC)
	}

	if cpu.SP != 0 {
		t.Fatalf("expected SP to be 0 after return, got %d", cpu.SP)
	}

	stepN(t, cpu, 1)

	if cpu.V[0] != 1 {
		t.Fatalf("expected V0 to be 1, got %d", cpu.V[0])
	}
}

func TestClearScreen(t *testing.T) {
	cpu := New()

	cpu.pixels[0] = true
	cpu.pixels[10] = true

	program := []byte{
		0x00, 0xE0, // clear screen
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 1)

	for i, pixel := range cpu.pixels {
		if pixel {
			t.Fatalf("expected display[%d] to be false", i)
		}
	}
}

func Test3XNNSkipIfVXEqualsNN(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0x0A, // V0 = 10
		0x30, 0x0A, // skip next if V0 == 10
		0x61, 0xFF, // skipped
		0x61, 0x05, // V1 = 5
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.V[1] != 5 {
		t.Fatalf("expected V1 to be 5, got %d", cpu.V[1])
	}
}

func Test4XNNSkipIfVXNotEqualsNN(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0x0A, // V0 = 10
		0x40, 0x05, // skip next if V0 != 5
		0x61, 0xFF, // skipped
		0x61, 0x05, // V1 = 5
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.V[1] != 5 {
		t.Fatalf("expected V1 to be 5, got %d", cpu.V[1])
	}
}

func Test5XY0SkipIfVXEqualsVY(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0x0A, // V0 = 10
		0x61, 0x0A, // V1 = 10
		0x50, 0x10, // skip next if V0 == V1
		0x62, 0xFF, // skipped
		0x62, 0x05, // V2 = 5
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 4)

	if cpu.V[2] != 5 {
		t.Fatalf("expected V2 to be 5, got %d", cpu.V[2])
	}
}

func Test8XY0SetVXToVY(t *testing.T) {
	cpu := New()

	program := []byte{
		0x61, 0x0A, // V1 = 10
		0x80, 0x10, // V0 = V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 2)

	if cpu.V[0] != 10 {
		t.Fatalf("expected V0 to be 10, got %d", cpu.V[0])
	}
}

func Test8XY1Or(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0b1010, // V0 = 10
		0x61, 0b1100, // V1 = 12
		0x80, 0x11, // V0 = V0 OR V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.V[0xF] = 1

	stepN(t, cpu, 3)

	if cpu.V[0] != 0b1110 {
		t.Fatalf("expected V0 to be 14, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be reset to 0, got %d", cpu.V[0xF])
	}
}

func Test8XY2And(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0b1010, // V0 = 10
		0x61, 0b1100, // V1 = 12
		0x80, 0x12, // V0 = V0 AND V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.V[0xF] = 1

	stepN(t, cpu, 3)

	if cpu.V[0] != 0b1000 {
		t.Fatalf("expected V0 to be 8, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be reset to 0, got %d", cpu.V[0xF])
	}
}

func Test8XY3Xor(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0b1010, // V0 = 10
		0x61, 0b1100, // V1 = 12
		0x80, 0x13, // V0 = V0 XOR V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.V[0xF] = 1

	stepN(t, cpu, 3)

	if cpu.V[0] != 0b0110 {
		t.Fatalf("expected V0 to be 6, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be reset to 0, got %d", cpu.V[0xF])
	}
}

func Test8XY1Or_ModernProfile(t *testing.T) {
	cpu := New()
	cpu.Quirks.LogicResetsVF = false

	program := []byte{
		0x60, 0b1010, // V0 = 10
		0x61, 0b1100, // V1 = 12
		0x80, 0x11, // V0 = V0 OR V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.V[0xF] = 1

	stepN(t, cpu, 3)

	if cpu.V[0] != 0b1110 {
		t.Fatalf("expected V0 to be 14, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 1 {
		t.Fatalf("expected VF to be left unchanged at 1, got %d", cpu.V[0xF])
	}
}

func Test8XY2And_ModernProfile(t *testing.T) {
	cpu := New()
	cpu.Quirks.LogicResetsVF = false

	program := []byte{
		0x60, 0b1010, // V0 = 10
		0x61, 0b1100, // V1 = 12
		0x80, 0x12, // V0 = V0 AND V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.V[0xF] = 1

	stepN(t, cpu, 3)

	if cpu.V[0] != 0b1000 {
		t.Fatalf("expected V0 to be 8, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 1 {
		t.Fatalf("expected VF to be left unchanged at 1, got %d", cpu.V[0xF])
	}
}

func Test8XY3Xor_ModernProfile(t *testing.T) {
	cpu := New()
	cpu.Quirks.LogicResetsVF = false

	program := []byte{
		0x60, 0b1010, // V0 = 10
		0x61, 0b1100, // V1 = 12
		0x80, 0x13, // V0 = V0 XOR V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.V[0xF] = 1

	stepN(t, cpu, 3)

	if cpu.V[0] != 0b0110 {
		t.Fatalf("expected V0 to be 6, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 1 {
		t.Fatalf("expected VF to be left unchanged at 1, got %d", cpu.V[0xF])
	}
}

func TestNewSuperChipProfile(t *testing.T) {
	profile := NewSuperChipProfile()

	expected := Quirks{
		WrapSprites:       false,
		ShiftModifiesVY:   false,
		StoreLoadMutatesI: false,
		JumpUsesVX:        true,
		IndexOverFlowsVF:  false,
		LogicResetsVF:     false,
	}

	if profile != expected {
		t.Fatalf("NewSuperChipProfile( = %+v, want %+v)", profile, expected)
	}
}

func Test8XY4AddNoCarry(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 10, // V0 = 10
		0x61, 20, // V1 = 20
		0x80, 0x14, // V0 = V0 + V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.V[0] != 30 {
		t.Fatalf("expected V0 to be 30, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be 0, got %d", cpu.V[0xF])
	}
}

func Test8XY4AddWithCarry(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 250, // V0 = 250
		0x61, 10, // V1 = 10
		0x80, 0x14, // V0 = V0 + V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.V[0] != 4 {
		t.Fatalf("expected V0 to wrap to 4, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 1 {
		t.Fatalf("expected VF to be 1, got %d", cpu.V[0xF])
	}
}

func Test8XY5SubtractNoBorrow(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 10, // V0 = 10
		0x61, 3, // V1 = 3
		0x80, 0x15, // V0 = V0 - V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.V[0] != 7 {
		t.Fatalf("expected V0 to be 7, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 1 {
		t.Fatalf("expected VF to be 1 because no borrow happened, got %d", cpu.V[0xF])
	}
}

func Test8XY5SubtractWithBorrow(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 3, // V0 = 3
		0x61, 10, // V1 = 10
		0x80, 0x15, // V0 = V0 - V1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.V[0] != 249 {
		t.Fatalf("expected V0 to wrap to 249, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be 0 because borrow happened, got %d", cpu.V[0xF])
	}
}

func Test8XY6ShiftRight(t *testing.T) {
	cpu := New()

	program := []byte{
		0x61, 0b00000101, // V1 = 5
		0x80, 0x16, // V0 = V1 >> 1, VF = dropped LSB
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 2)

	if cpu.V[0] != 0b00000010 {
		t.Fatalf("expected V0 to be 2, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 1 {
		t.Fatalf("expected VF to be 1, got %d", cpu.V[0xF])
	}
}

func Test8XY6ShiftRight_ModernProfile(t *testing.T) {
	cpu := New()
	cpu.Quirks.ShiftModifiesVY = false

	program := []byte{
		0x60, 0b00000110, // V0 = 6 (VX)
		0x61, 0b00000101, // V1 = 5 (VY) - deliberately different from V0
		0x80, 0x16, // V0 = V0 >> 1, VF = dropped LSB
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	// If the source were VY (5) instead of VX (6), V0 would be 2 and VF would be 1.
	if cpu.V[0] != 0b00000011 {
		t.Fatalf("expected V0 to be 3 (shifted from VX), got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be 0 (dropped bit from VX), got %d", cpu.V[0xF])
	}
}

func Test8XY7ReverseSubtract(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 3, // V0 = 3
		0x61, 10, // V1 = 10
		0x80, 0x17, // V0 = V1 - V0
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.V[0] != 7 {
		t.Fatalf("expected V0 to be 7, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 1 {
		t.Fatalf("expected VF to be 1 because no borrow happened, got %d", cpu.V[0xF])
	}
}

func Test8XYEShiftLeft(t *testing.T) {
	cpu := New()

	program := []byte{
		0x61, 0b10000001, // V1 = 129
		0x80, 0x1E, // V0 = V1 << 1, VF = dropped MSB
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 2)

	if cpu.V[0] != 0b00000010 {
		t.Fatalf("expected V0 to be 2, got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 1 {
		t.Fatalf("expected VF to be 1, got %d", cpu.V[0xF])
	}
}

func Test8XYEShiftLeft_ModernProfile(t *testing.T) {
	cpu := New()
	cpu.Quirks.ShiftModifiesVY = false

	program := []byte{
		0x60, 0b01000010, // V0 = 66 (VX)
		0x61, 0b10000001, // V1 = 129 (VY) - deliberately different from V0
		0x80, 0x1E, // V0 = V0 << 1, VF = dropped MSB
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	// If the source were VY (129) instead of VX (66), V0 would be 2 and VF would be 1.
	if cpu.V[0] != 0b10000100 {
		t.Fatalf("expected V0 to be 132 (shifted from VX), got %d", cpu.V[0])
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be 0 (dropped bit from VX), got %d", cpu.V[0xF])
	}
}

func Test9XY0SkipIfVXNotEqualsVY(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 1, // V0 = 1
		0x61, 2, // V1 = 2
		0x90, 0x10, // skip next if V0 != V1
		0x62, 99, // skipped
		0x62, 42, // V2 = 42
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 4)

	if cpu.V[2] != 42 {
		t.Fatalf("expected V2 to be 42, got %d", cpu.V[2])
	}
}

func TestANNNSetIndexRegister(t *testing.T) {
	cpu := New()

	program := []byte{
		0xA3, 0x00, // I = 0x300
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 1)

	if cpu.I != 0x300 {
		t.Fatalf("expected I to be 0x300, got 0x%03X", cpu.I)
	}
}

func TestBNNNJumpToV0PlusNNN(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0x04, // V0 = 4
		0xB2, 0x08, // PC = 0x208 + V0 = 0x20C
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 2)

	if cpu.PC != 0x20C {
		t.Fatalf("expected PC to be 0x20C, got 0x%03X", cpu.PC)
	}
}

func TestBXNNJumpToVXPlusNNN_JumpUsesVX(t *testing.T) {
	cpu := New()
	cpu.Quirks.JumpUsesVX = true

	program := []byte{
		0x60, 0x04, // V0 = 4 (deliberately different from V2)
		0x62, 0x10, // V2 = 16
		0xB2, 0x08, // PC = nnn(0x208) + V2 = 0x218 (register selected by opcode nibble, not V0)
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	// If the jump used V0 (4) instead of the opcode-selected V2 (16), PC would be 0x20C.
	if cpu.PC != 0x218 {
		t.Fatalf("expected PC to be 0x218 (nnn + V2), got 0x%03X", cpu.PC)
	}
}

func TestCXNNRandomAndMask(t *testing.T) {
	cpu := New()

	program := []byte{
		0xC0, 0x0F, // V0 = random byte AND 0x0F
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 1)

	if cpu.V[0] > 0x0F {
		t.Fatalf("expected V0 to be masked to <= 0x0F, got 0x%02X", cpu.V[0])
	}
}

// internal/chip8/cpu_test.go

func TestDXYNDrawSinglePixelSprite(t *testing.T) {
	cpu := New()

	program := []byte{
		0xA3, 0x00, // I = 0x300
		0x60, 10, // V0 = x = 10
		0x61, 5, // V1 = y = 5
		0xD0, 0x11, // draw sprite at (V0, V1), height 1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.Memory[0x300] = 0b10000000

	stepN(t, cpu, 4)

	index := 5*MaxScreenWidth + 10

	if !cpu.pixels[index] {
		t.Fatalf("expected pixel at (10, 5) to be on")
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be 0 when no collision happens, got %d", cpu.V[0xF])
	}
}

func TestDXYNDrawFullByteSprite(t *testing.T) {
	cpu := New()

	program := []byte{
		0xA3, 0x00, // I = 0x300
		0x60, 4, // V0 = x = 4
		0x61, 3, // V1 = y = 3
		0xD0, 0x11, // draw sprite at (V0, V1), height 1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.Memory[0x300] = 0b11110000

	stepN(t, cpu, 4)

	onPixels := []int{
		3*MaxScreenWidth + 4,
		3*MaxScreenWidth + 5,
		3*MaxScreenWidth + 6,
		3*MaxScreenWidth + 7,
	}

	offPixels := []int{
		3*MaxScreenWidth + 8,
		3*MaxScreenWidth + 9,
		3*MaxScreenWidth + 10,
		3*MaxScreenWidth + 11,
	}

	for _, index := range onPixels {
		if !cpu.pixels[index] {
			t.Fatalf("expected display[%d] to be on", index)
		}
	}

	for _, index := range offPixels {
		if cpu.pixels[index] {
			t.Fatalf("expected display[%d] to be off", index)
		}
	}
}

func TestDXYNDrawMultiRowSprite(t *testing.T) {
	cpu := New()

	program := []byte{
		0xA3, 0x00, // I = 0x300
		0x60, 2, // V0 = x = 2
		0x61, 4, // V1 = y = 4
		0xD0, 0x13, // draw sprite at (V0, V1), height 3
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.Memory[0x300] = 0b10000000
	cpu.Memory[0x301] = 0b01000000
	cpu.Memory[0x302] = 0b00100000

	stepN(t, cpu, 4)

	expectedPixels := []int{
		4*MaxScreenWidth + 2,
		5*MaxScreenWidth + 3,
		6*MaxScreenWidth + 4,
	}

	for _, index := range expectedPixels {
		if !cpu.pixels[index] {
			t.Fatalf("expected display[%d] to be on", index)
		}
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be 0 when no collision happens, got %d", cpu.V[0xF])
	}
}

func TestDXYNDrawSpriteCollision(t *testing.T) {
	cpu := New()

	program := []byte{
		0xA3, 0x00, // I = 0x300
		0x60, 10, // V0 = x = 10
		0x61, 5, // V1 = y = 5
		0xD0, 0x11, // draw once
		0xD0, 0x11, // draw same sprite again
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.Memory[0x300] = 0b10000000

	stepN(t, cpu, 4)

	index := 5*MaxScreenWidth + 10

	if !cpu.pixels[index] {
		t.Fatalf("expected pixel to be on after first draw")
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be 0 after first draw, got %d", cpu.V[0xF])
	}

	stepN(t, cpu, 1)

	if cpu.pixels[index] {
		t.Fatalf("expected pixel to be off after second draw")
	}

	if cpu.V[0xF] != 1 {
		t.Fatalf("expected VF to be 1 after collision, got %d", cpu.V[0xF])
	}
}

func TestDXYNDrawSpriteWrapsAroundScreen(t *testing.T) {
	cpu := New()
	cpu.Quirks.WrapSprites = true

	program := []byte{
		0xA3, 0x00, // I = 0x300
		0x60, 63, // V0 = x = 63
		0x61, 31, // V1 = y = 31
		0xD0, 0x12, // draw sprite at bottom-right, height 2
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.Memory[0x300] = 0b11000000
	cpu.Memory[0x301] = 0b10000000

	stepN(t, cpu, 4)

	expectedPixels := []int{
		31*MaxScreenWidth + 63, // first row, x = 63
		31*MaxScreenWidth + 0,  // first row wraps to x = 0
		0*MaxScreenWidth + 63,  // second row wraps to y = 0
	}

	for _, index := range expectedPixels {
		if !cpu.pixels[index] {
			t.Fatalf("expected display[%d] to be on", index)
		}
	}
}

func TestDXYNReturnsErrorWhenSpriteMemoryIsOutOfBounds(t *testing.T) {
	cpu := New()

	cpu.I = MemorySize - 1

	if err := cpu.Execute(0xD002); err == nil {
		t.Fatal("expected drawing a sprite past memory bounds to return an error")
	}
}

func TestEX9ESkipIfKeyInVXIsPressed(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0x0A, // V0 = key A
		0xE0, 0x9E, // skip next if key in V0 is pressed
		0x61, 0xFF, // skipped
		0x61, 0x05, // V1 = 5
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.Keys[0xA] = true

	stepN(t, cpu, 3)

	if cpu.V[1] != 5 {
		t.Fatalf("expected V1 to be 5, got %d", cpu.V[1])
	}
}

func TestEXA1SkipIfKeyInVXIsNotPressed(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0x05, // V0 = key 5
		0xE0, 0xA1, // skip next if key in V0 is not pressed
		0x61, 0xFF, // skipped
		0x61, 0x05, // V1 = 5
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.V[1] != 5 {
		t.Fatalf("expected V1 to be 5, got %d", cpu.V[1])
	}
}

func TestFX07ReadDelayTimer(t *testing.T) {
	cpu := New()

	program := []byte{
		0xF1, 0x07, // V1 = delay timer
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.DelayTimer = 12

	stepN(t, cpu, 1)

	if cpu.V[1] != 12 {
		t.Fatalf("expected V1 to be 12, got %d", cpu.V[1])
	}
}

func TestFX0AWaitsUntilKeyPress(t *testing.T) {
	cpu := New()

	program := []byte{
		0xF2, 0x0A, // wait for key, store in V2
		0x60, 0x01, // V0 = 1
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 1)

	if cpu.PC != ProgramStart {
		t.Fatalf("expected PC to stay at 0x%03X while waiting, got 0x%03X", ProgramStart, cpu.PC)
	}

	if cpu.V[2] != 0 {
		t.Fatalf("expected V2 to remain 0 while no key is pressed, got %d", cpu.V[2])
	}
}

func TestFX0AWaitsForPressedKeyToBeReleased(t *testing.T) {
	cpu := New()

	program := []byte{
		0xF2, 0x0A, // wait for key, store in V2
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.Keys[0xC] = true

	stepN(t, cpu, 1)

	if cpu.V[2] != 0 {
		t.Fatalf("expected V2 to remain 0 until key is released, got 0x%X", cpu.V[2])
	}

	if cpu.PC != ProgramStart {
		t.Fatalf("expected PC to stay at 0x%03X while key is held, got 0x%03X", ProgramStart, cpu.PC)
	}
}

func TestFX0AStoresPressedKeyAfterRelease(t *testing.T) {
	cpu := New()

	program := []byte{
		0xF2, 0x0A, // wait for key, store in V2
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.Keys[0xC] = true
	stepN(t, cpu, 1)

	cpu.Keys[0xC] = false
	stepN(t, cpu, 1)

	if cpu.V[2] != 0xC {
		t.Fatalf("expected V2 to store key 0xC, got 0x%X", cpu.V[2])
	}

	if cpu.PC != ProgramStart+2 {
		t.Fatalf("expected PC to advance to 0x%03X, got 0x%03X", ProgramStart+2, cpu.PC)
	}
}

func TestFX15AndFX18SetTimers(t *testing.T) {
	cpu := New()

	program := []byte{
		0x62, 0x09, // V2 = 9
		0xF2, 0x15, // delay timer = V2
		0xF2, 0x18, // sound timer = V2
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.DelayTimer != 9 {
		t.Fatalf("expected DelayTimer to be 9, got %d", cpu.DelayTimer)
	}

	if cpu.SoundTimer != 9 {
		t.Fatalf("expected SoundTimer to be 9, got %d", cpu.SoundTimer)
	}
}

func TestFX1EAddVXToI(t *testing.T) {
	cpu := New()

	program := []byte{
		0xA3, 0x00, // I = 0x300
		0x60, 0x05, // V0 = 5
		0xF0, 0x1E, // I += V0
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.I != 0x305 {
		t.Fatalf("expected I to be 0x305, got 0x%03X", cpu.I)
	}
}

func TestFX1ESetsVFOnOverflow_IndexOverFlowsVF(t *testing.T) {
	cpu := New()
	cpu.Quirks.IndexOverFlowsVF = true

	cpu.I = 0x0FFF
	cpu.V[0] = 5

	if err := cpu.Execute(0xF01E); err != nil {
		t.Fatal(err)
	}

	if cpu.I != 0x1004 {
		t.Fatalf("expected I to be 0x1004, got 0x%03X", cpu.I)
	}

	if cpu.V[0xF] != 1 {
		t.Fatalf("expected VF to be 1 on overflow, got %d", cpu.V[0xF])
	}
}

func TestFX1EClearsVFWhenNoOverflow_IndexOverFlowsVF(t *testing.T) {
	cpu := New()
	cpu.Quirks.IndexOverFlowsVF = true

	cpu.I = 0x0100
	cpu.V[0] = 5

	if err := cpu.Execute(0xF01E); err != nil {
		t.Fatal(err)
	}

	if cpu.I != 0x0105 {
		t.Fatalf("expected I to be 0x105, got 0x%03X", cpu.I)
	}

	if cpu.V[0xF] != 0 {
		t.Fatalf("expected VF to be 0 when no overflow happens, got %d", cpu.V[0xF])
	}
}

func TestFX1ELeavesVFUntouchedWhenQuirkDisabled(t *testing.T) {
	cpu := New()
	cpu.Quirks.IndexOverFlowsVF = false

	cpu.V[0xF] = 7 // neither 0 nor 1, so it can't be confused with an on-branch outcome
	cpu.I = 0x0FFF
	cpu.V[0] = 5

	if err := cpu.Execute(0xF01E); err != nil {
		t.Fatal(err)
	}

	if cpu.I != 0x1004 {
		t.Fatalf("expected I to be 0x1004, got 0x%03X", cpu.I)
	}

	if cpu.V[0xF] != 7 {
		t.Fatalf("expected VF to remain untouched at 7, got %d", cpu.V[0xF])
	}
}

func TestFX29SetIToFontSprite(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0x0A, // V0 = A
		0xF0, 0x29, // I = font address for A
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 2)

	expected := FontStart + uint16(0xA)*5
	if cpu.I != expected {
		t.Fatalf("expected I to be 0x%03X, got 0x%03X", expected, cpu.I)
	}
}

func TestFX33StoreBCD(t *testing.T) {
	cpu := New()

	program := []byte{
		0xA3, 0x00, // I = 0x300
		0x60, 123, // V0 = 123
		0xF0, 0x33, // store BCD of V0 at I, I+1, I+2
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 3)

	if cpu.Memory[0x300] != 1 || cpu.Memory[0x301] != 2 || cpu.Memory[0x302] != 3 {
		t.Fatalf(
			"expected BCD bytes [1 2 3], got [%d %d %d]",
			cpu.Memory[0x300],
			cpu.Memory[0x301],
			cpu.Memory[0x302],
		)
	}
}

func TestFX55StoreRegisters(t *testing.T) {
	cpu := New()

	program := []byte{
		0x60, 0x01, // V0 = 1
		0x61, 0x02, // V1 = 2
		0x62, 0x03, // V2 = 3
		0xA3, 0x00, // I = 0x300
		0xF2, 0x55, // store V0 through V2 at I
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	stepN(t, cpu, 5)

	for offset, expected := range []byte{1, 2, 3} {
		if cpu.Memory[0x300+offset] != expected {
			t.Fatalf("expected memory[0x%03X] to be %d, got %d", 0x300+offset, expected, cpu.Memory[0x300+offset])
		}
	}
}

func TestFX65LoadRegisters(t *testing.T) {
	cpu := New()

	program := []byte{
		0xA3, 0x00, // I = 0x300
		0xF2, 0x65, // load V0 through V2 from I
	}

	if err := cpu.LoadProgram(program); err != nil {
		t.Fatal(err)
	}

	cpu.Memory[0x300] = 4
	cpu.Memory[0x301] = 5
	cpu.Memory[0x302] = 6

	stepN(t, cpu, 2)

	for register, expected := range []byte{4, 5, 6} {
		if cpu.V[register] != expected {
			t.Fatalf("expected V%d to be %d, got %d", register, expected, cpu.V[register])
		}
	}
}

func TestUpdateTimersDecrementsNonZeroTimers(t *testing.T) {
	cpu := New()

	cpu.DelayTimer = 2
	cpu.SoundTimer = 1

	cpu.UpdateTimers()

	if cpu.DelayTimer != 1 {
		t.Fatalf("expected DelayTimer to be 1, got %d", cpu.DelayTimer)
	}

	if cpu.SoundTimer != 0 {
		t.Fatalf("expected SoundTimer to be 0, got %d", cpu.SoundTimer)
	}

	cpu.UpdateTimers()

	if cpu.DelayTimer != 0 {
		t.Fatalf("expected DelayTimer to be 0, got %d", cpu.DelayTimer)
	}

	if cpu.SoundTimer != 0 {
		t.Fatalf("expected SoundTimer to stay 0, got %d", cpu.SoundTimer)
	}
}
