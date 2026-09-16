package axonvm

import (
	"bytes"
	"strings"
	"testing"
)

// The block-versus-single-line `If` distinction at an ASP tag boundary.
//
// VBScript's rule: `If cond Then <statement>` on one line is a COMPLETE
// single-line If. Under ASP a `%>` ends the statement, so a following `End If`
// belongs to an enclosing block If — or to nothing at all, which is an error.
// `ElseIf cond Then <statement>` is different: it sits inside an already-open
// block, so the block stays open and `End If` closes it.
//
// Every expectation below was measured on IIS 10.0 / VBScript 5.8.16384 on
// 2026-09-16, one construct per page, reading the HTTP status of each:
// 200 with a body means accepted, 500 means rejected.
//
// Upstream v2.3.20 inverted both halves of this — it made the single-line form
// absorb a trailing `End If` across a boundary, and left the `ElseIf` form
// (guimaraeslucas/axonasp#124) rejected. Reported as guimaraeslucas/axonasp#129.

func compileAndRun(t *testing.T, source string) (string, error) {
	t.Helper()
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		return "", err
	}
	vm := NewVMFromCompiler(compiler)
	host := NewMockHost()
	var out bytes.Buffer
	host.SetOutput(&out)
	vm.SetHost(host)
	if err := vm.Run(); err != nil {
		return out.String(), err
	}
	host.Response().Flush()
	return out.String(), nil
}

// A single-line If must not swallow the End If that closes the block around it.
func TestSingleLineIfDoesNotAbsorbEndIfAcrossTagBoundary(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			// IIS: 200, "As". The inner If completes at the boundary, so the
			// End If closes the outer block.
			name:   "inner condition true",
			source: `<% dim x : x = 1 %><% if x = 1 then %>A<% if x = 1 then Response.Write "s" %><% end if %>`,
			want:   "As",
		},
		{
			// IIS: 200, "A".
			name:   "inner condition false",
			source: `<% dim x : x = 1 %><% if x = 1 then %>A<% if x = 0 then Response.Write "s" %><% end if %>`,
			want:   "A",
		},
		{
			// IIS: 200, "xTAIL". No End If anywhere; never affected, pinned so a
			// future fix to the rule above cannot quietly break it.
			name: "single-line If with no End If at all",
			source: `<% dim x : x = 1
if x = 1 then Response.Write "x" %>TAIL`,
			want: "xTAIL",
		},
		{
			// Same-line End If after a colon is genuine VBScript and stays legal.
			name: "trailing End If on the same line after a colon",
			source: `<% dim x : x = 1
if x = 1 then Response.Write "y" : end if
Response.Write "Z" %>`,
			want: "yZ",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := compileAndRun(t, tc.source)
			if err != nil {
				t.Fatalf("compile/run failed: %v", err)
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("want output containing %q, got %q", tc.want, got)
			}
		})
	}
}

// The orphan form: a single-line If, a boundary, then an End If with no
// enclosing block. IIS returns HTTP 500, so this must not compile.
func TestOrphanEndIfAfterSingleLineIfIsRejected(t *testing.T) {
	source := `<% dim x : x = 1
if x = 1 then Response.Write "x" %><% end if %>`
	if _, err := compileAndRun(t, source); err == nil {
		t.Fatal("expected a compilation error for an orphan End If after a single-line If, got none")
	}
}

// guimaraeslucas/axonasp#124: a block If whose last arm is an inline ElseIf,
// closed across a tag boundary. IIS: 200, "one reached the end".
func TestBlockIfEndsAcrossTagBoundaryAfterInlineElseIf(t *testing.T) {
	source := `<%
dim x : x = 1
if x = 0 then
	Response.Write "WRONG branch"
elseif x = 1 then Response.Write "one" %><% end if
Response.Write " reached the end"
%>`
	got, err := compileAndRun(t, source)
	if err != nil {
		t.Fatalf("compile/run failed: %v", err)
	}
	if want := "one reached the end"; !strings.Contains(got, want) {
		t.Errorf("want output containing %q, got %q", want, got)
	}
	if strings.Contains(got, "WRONG branch") {
		t.Errorf("the ElseIf arm did not branch correctly: %q", got)
	}
}

// NOT COVERED: real HTML between an inline ElseIf arm and the End If, e.g.
//   elseif x = 1 then Response.Write "one" %>MIDDLE<% end if
// consumeTagBoundaryKeyword deliberately refuses to consume the boundary when
// non-whitespace HTML intervenes, so this does not compile. Upstream v2.3.19
// and v2.3.22 do not compile it either, so the fork loses nothing. It is not
// asserted here because it has not been measured on IIS — the singleline-if
// probe in the portal repo carries an "htmlbetween" page for the next run.
