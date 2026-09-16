package axonvm

import "testing"

// opcodeOperandSize is called by the optimiser while walking bytecode, so it can
// be handed an instruction whose operand bytes run past the end of the slice.
// Reading them unchecked panicked the entire compile with
// "index out of range [1] with length 1", surfacing as a page that simply would
// not compile. Found on payroll/make-invoices.asp in the PGS portal, which
// compiles on v2.3.19 and panics from v2.3.20.
func TestOpcodeOperandSizeTruncated(t *testing.T) {
	tests := []struct {
		name     string
		op       OpCode
		bytecode []byte
		ip       int
	}{
		{"ExtPrefix with no operand byte", OpExtPrefix, []byte{byte(OpExtPrefix)}, 0},
		{"JSObjectRest with no count bytes", OpJSObjectRest, []byte{byte(OpJSObjectRest)}, 0},
		{"JSObjectRest with one count byte", OpJSObjectRest, []byte{byte(OpJSObjectRest), 0x00}, 0},
		{"ExtPrefix at the tail of a longer run", OpExtPrefix,
			[]byte{byte(OpNop), byte(OpNop), byte(OpExtPrefix)}, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("opcodeOperandSize panicked on truncated bytecode: %v", r)
				}
			}()
			got := opcodeOperandSize(tc.op, tc.bytecode, tc.ip)
			if got < 0 {
				t.Errorf("negative operand size %d", got)
			}
			// A walker does ip += 1 + size; that must not run backwards.
			if tc.ip+1+got < tc.ip {
				t.Errorf("size %d would move the walker backwards", got)
			}
		})
	}
}

// A full peephole pass over bytecode ending in a truncated extended opcode must
// not take the compiler down either — that is the path the real page hit.
func TestPeepholePassSurvivesTruncatedTailInstruction(t *testing.T) {
	c := &Compiler{}
	c.bytecode = []byte{
		byte(OpConstant), 0x00, 0x00,
		byte(OpConstant), 0x00, 0x01,
		byte(OpNop),
		byte(OpExtPrefix), // operand byte deliberately missing
	}
	c.constants = []Value{NewString("a"), NewString("b")}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("optimizePeepholePass panicked: %v", r)
		}
	}()
	c.optimizePeepholePass()
}
