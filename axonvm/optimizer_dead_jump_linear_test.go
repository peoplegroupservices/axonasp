package axonvm

import (
	"bytes"
	"testing"
)

// referenceDeadConditionalJumpPass is optimizeDeadConditionalJumpPass exactly as
// it was written before the linear rewrite, kept here as a test oracle. It finds
// the preceding instruction by rescanning the bytecode from offset zero, which is
// what made the pass quadratic; the behaviour it defines is the behaviour the
// rewrite has to preserve.
func referenceDeadConditionalJumpPass(c *Compiler) bool {
	if c == nil || len(c.bytecode) == 0 {
		return false
	}

	targets := collectJumpTargets(c.bytecode)
	changed := false

	for ip := 0; ip < len(c.bytecode); {
		op := OpCode(c.bytecode[ip])
		size := opcodeOperandSize(op, c.bytecode, ip)
		instrEnd := ip + 1 + size
		if instrEnd > len(c.bytecode) {
			break
		}

		if op != OpJumpIfFalse {
			ip = instrEnd
			continue
		}

		target := int(bigEndianUint32(c.bytecode[ip+1 : ip+5]))
		if target <= instrEnd || target > len(c.bytecode) || target <= ip {
			ip = instrEnd
			continue
		}

		condStart := findPreviousInstructionStart(c.bytecode, ip)
		for condStart >= 0 && OpCode(c.bytecode[condStart]) == OpNop {
			condStart = findPreviousInstructionStart(c.bytecode, condStart)
		}
		if condStart < 0 || OpCode(c.bytecode[condStart]) != OpConstant || condStart+3 > len(c.bytecode) {
			ip = instrEnd
			continue
		}

		constIdx := int(bigEndianUint16(c.bytecode[condStart+1 : condStart+3]))
		if constIdx < 0 || constIdx >= len(c.constants) {
			ip = instrEnd
			continue
		}
		if !isCompileTimeFalseValue(c.constants[constIdx]) {
			ip = instrEnd
			continue
		}

		if hasTargetInRange(targets, instrEnd, target-1) {
			ip = instrEnd
			continue
		}

		mutated := false
		for p := instrEnd; p < target; p++ {
			if OpCode(c.bytecode[p]) != OpNop {
				c.bytecode[p] = byte(OpNop)
				mutated = true
			}
		}
		if mutated {
			changed = true
		}
		ip = target
	}

	return changed
}

func bigEndianUint32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func bigEndianUint16(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])
}

// deadJumpSources are shaped to actually produce dead conditional jumps, so the
// comparison below is not vacuous.
var deadJumpSources = []string{
	`<% if false then Response.Write "no" end if : Response.Write "yes" %>`,
	`<% dim x : x = 0
if x then
	Response.Write "no"
else
	Response.Write "yes"
end if %>`,
	`<% if "" then Response.Write "no" end if
if 0 then Response.Write "no2" end if
Response.Write "done" %>`,
	`<% dim i
for i = 1 to 3
	if false then
		Response.Write "never"
	elseif i = 2 then
		Response.Write "two"
	end if
next %>`,
	`<% if false then %>HTML<% end if %>tail`,
	`<% sub s()
	if false then Response.Write "no" end if
end sub
s() %>`,
}

// TestDeadConditionalJumpPassMatchesRescan pins the linear rewrite to the
// original rescanning definition. It compares WHOLE COMPILES rather than one
// pass over already-optimised bytecode, because by the time Compile returns the
// dead jumps are gone and re-running the pass would prove nothing.
func TestDeadConditionalJumpPassMatchesRescan(t *testing.T) {
	compileWith(t, nil) // sanity: the hook is restored between sub-tests

	sawDifferenceOpportunity := false
	for i, src := range deadJumpSources {
		fast := compileWith(t, (*Compiler).optimizeDeadConditionalJumpPass)
		ref := compileWith(t, referenceDeadConditionalJumpPass)

		gotCode, gotOut := fast(src)
		wantCode, wantOut := ref(src)

		if !bytes.Equal(gotCode, wantCode) {
			t.Errorf("source %d: compiled bytecode differs from the reference implementation", i)
		}
		if gotOut != wantOut {
			t.Errorf("source %d: output %q, reference %q", i, gotOut, wantOut)
		}
		if bytes.IndexByte(gotCode, byte(OpNop)) >= 0 {
			sawDifferenceOpportunity = true
		}
	}

	if !sawDifferenceOpportunity {
		t.Fatal("no source produced any OpNop — the corpus never exercised the pass")
	}
}

// compileWith returns a compile function that runs with the given dead-jump pass
// implementation installed, restoring the default when the test ends.
func compileWith(t *testing.T, impl func(*Compiler) bool) func(string) ([]byte, string) {
	t.Helper()
	original := deadConditionalJumpPass
	if impl != nil {
		deadConditionalJumpPass = impl
	}
	t.Cleanup(func() { deadConditionalJumpPass = original })

	return func(src string) ([]byte, string) {
		t.Helper()
		c := NewASPCompiler(src)
		if err := c.Compile(); err != nil {
			t.Fatalf("compile failed: %v", err)
		}
		code := append([]byte(nil), c.bytecode...)

		vm := NewVMFromCompiler(c)
		host := NewMockHost()
		var out bytes.Buffer
		host.SetOutput(&out)
		vm.SetHost(host)
		if err := vm.Run(); err != nil {
			t.Fatalf("run failed: %v", err)
		}
		host.Response().Flush()
		return code, out.String()
	}
}

// BenchmarkCompileDeadJumpHeavy documents the complexity fix. Before it, work grew
// quadratically with bytecode length: compiling a 48k-line source spent 95% of its
// time in this pass, and one PGS portal include tree took ~9.5s on its own.
func BenchmarkCompileDeadJumpHeavy(b *testing.B) {
	for _, n := range []int{200, 400, 800, 1600} {
		src := buildDeadJumpSource(n)
		b.Run(itoaSmall(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				c := NewASPCompiler(src)
				if err := c.Compile(); err != nil {
					b.Fatalf("compile failed: %v", err)
				}
			}
		})
	}
}

func buildDeadJumpSource(n int) string {
	var b bytes.Buffer
	b.WriteString("<%\n")
	for i := 0; i < n; i++ {
		b.WriteString("if false then\n\tResponse.Write \"never\"\nend if\n")
		b.WriteString("dim v" + itoaSmall(i) + " : v" + itoaSmall(i) + " = " + itoaSmall(i) + "\n")
	}
	b.WriteString("%>")
	return b.String()
}

func itoaSmall(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [8]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
