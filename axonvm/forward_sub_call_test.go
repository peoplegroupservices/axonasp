package axonvm

import (
	"bytes"
	"strings"
	"testing"
)

// A bare Sub call made from inside a procedure body, to a Sub declared later in
// the file, used to compile to a load of a procedure LOCAL rather than the Sub.
// Nothing registered a forward-call patch for that load, so at runtime the call
// target was an empty local: the call did nothing at all, raised no error, and
// returned no value. Statements simply vanished.
//
// Found in the PGS portal, where inc/_global.asp calls cleanOpen (declared at
// line 5616) from inside getSessionID2 (line 4565). Signing in silently failed
// to create a session: the INSERT was never sent, the recordset stayed closed,
// and the page reported "Failed to create record" with no error anywhere.
// 98 of the 100 call sites in that file are forward references.
func TestForwardSubCallFromInsideProcedure(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "forward Sub called from inside a Function",
			source: `<%
function caller()
	target 1, "payload", 3, 4, 5, 6
	caller = "ret"
end function
sub target(a, b, c, d, e, f)
	Response.Write "called:" & b
end sub
Response.Write caller()
%>`,
			want: "called:payload",
		},
		{
			name: "forward Sub called from inside a Sub",
			source: `<%
sub caller()
	target "payload"
end sub
sub target(b)
	Response.Write "called:" & b
end sub
caller
%>`,
			want: "called:payload",
		},
		{
			name: "forward zero-argument Sub called from inside a Function",
			source: `<%
function caller()
	target
	caller = ""
end function
sub target()
	Response.Write "called:zero"
end sub
Response.Write caller()
%>`,
			want: "called:zero",
		},
		{
			name: "backward reference still works",
			source: `<%
sub target(b)
	Response.Write "called:" & b
end sub
function caller()
	target "payload"
	caller = ""
end function
Response.Write caller()
%>`,
			want: "called:payload",
		},
		{
			// A real local of the same name must keep winning, or this fix would
			// break ordinary code.
			name: "a local variable shadowing the name is not rewritten",
			source: `<%
function caller()
	dim target
	target = "i am a local"
	Response.Write "local:" & target
	caller = ""
end function
sub target(b)
	Response.Write "WRONG: the Sub was called"
end sub
Response.Write caller()
%>`,
			want: "local:i am a local",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewASPCompiler(tc.source)
			if err := c.Compile(); err != nil {
				t.Fatalf("compile failed: %v", err)
			}
			vm := NewVMFromCompiler(c)
			host := NewMockHost()
			var out bytes.Buffer
			host.SetOutput(&out)
			vm.SetHost(host)
			if err := vm.Run(); err != nil {
				t.Fatalf("run failed: %v", err)
			}
			host.Response().Flush()

			got := out.String()
			if !strings.Contains(got, tc.want) {
				t.Errorf("want output containing %q, got %q", tc.want, got)
			}
			if strings.Contains(got, "WRONG") {
				t.Errorf("a Sub was called where a local variable should have won: %q", got)
			}
		})
	}
}
