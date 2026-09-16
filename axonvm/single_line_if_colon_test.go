/*
 * AxonASP Server
 * Copyright (C) 2026 G3pix Ltda. All rights reserved.
 *
 * Developed by Lucas Guimarães - G3pix Ltda
 * Contact: https://g3pix.com.br
 * Project URL: https://g3pix.com.br/axonasp
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 *
 * Attribution Notice:
 * If this software is used in other projects, the name "AxonASP Server"
 * must be cited in the documentation or "About" section.
 *
 * Contribution Policy:
 * Modifications to the core source code of AxonASP Server must be
 * made available under this same license terms.
 */
package axonvm

import "testing"

// TestSingleLineIfColonReproduction validates the reproduction case where a colon
// follows 'Then' in a single-line If statement without requiring End If.
func TestSingleLineIfColonReproduction(t *testing.T) {
	source := `<%
Dim a, b
a = "x"
if a <> "" then : b = "set" : a = "done"
Response.Write "b=" & b & " a=" & a
%>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := "b=set a=done"
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}

// TestSingleLineIfColonMultipleStatements verifies multiple colon-separated statements on single-line If.
func TestSingleLineIfColonMultipleStatements(t *testing.T) {
	source := `<%
If True Then : Response.Write "A" : Response.Write "B"
%>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := "AB"
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}

// TestSingleLineIfColonVariableWhitespace verifies single-line If with variable whitespace before/after colon.
func TestSingleLineIfColonVariableWhitespace(t *testing.T) {
	source := `<%
Dim a
a = 0
If True Then   :   a = 1
Response.Write "a=" & a
%>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := "a=1"
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}

// TestBlockIfWithColonsInStatements verifies that block If statements containing colons in inner statements
// continue to parse and execute properly as blocks without regression.
func TestBlockIfWithColonsInStatements(t *testing.T) {
	source := `<%
Dim a, b, c, d
a = 0 : b = 0 : c = 0 : d = 0
If True Then
    a = 10 : b = 20
    If True Then : c = 30 : d = 40
    Response.Write "a=" & a & " b=" & b & " c=" & c & " d=" & d
End If
%>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := "a=10 b=20 c=30 d=40"
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}

// ---------------------------------------------------------------------------
// FORK DIVERGENCE — see FORK.md and guimaraeslucas/axonasp#129.
//
// The three tests below asserted, upstream, that a single-line
// `If cond Then <stmt>` followed by `%><% End If %>` compiles and runs the
// statement. Measured on IIS 10.0 / VBScript 5.8.16384 on 2026-09-16, IIS
// REJECTS that source with HTTP 500: `If cond Then <stmt>` is a complete
// single-line If, the tag boundary ends it, and the `End If` is an orphan.
//
// The colon-separated and multi-line spellings behave identically, so the
// measurement covers the exact sources used here. The expectations are
// therefore inverted in this fork. If upstream resolves #129 the other way,
// this is the file where the disagreement lives.
// ---------------------------------------------------------------------------

// TestSingleLineIfTagBoundaryReproduction pins that an orphan End If after a
// single-line If is rejected, as IIS rejects it.
func TestSingleLineIfTagBoundaryReproduction(t *testing.T) {
	source := `<% Dim x : x = 1 : If x = 1 Then Response.Write "one" %><% End If %>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err == nil {
		t.Fatal("expected a compilation error for an orphan End If after a single-line If, got none")
	}
}

// TestSingleLineIfTagBoundaryFalseBranch pins the same rejection when the
// condition is false — it is a compile-time question, not a branch one.
func TestSingleLineIfTagBoundaryFalseBranch(t *testing.T) {
	source := `<% Dim x : x = 2 : If x = 1 Then Response.Write "one" %><% End If %>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err == nil {
		t.Fatal("expected a compilation error for an orphan End If after a single-line If, got none")
	}
}

// TestSingleLineIfTagBoundaryWithWhitespaceAndNewlines pins that whitespace and
// newlines between the tags do not change the answer: it is the construct that
// is rejected, not the tag adjacency.
func TestSingleLineIfTagBoundaryWithWhitespaceAndNewlines(t *testing.T) {
	source := `<%
Dim x : x = 1
If x = 1 Then Response.Write "one" %>
   
<% End If %>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err == nil {
		t.Fatal("expected a compilation error for an orphan End If after a single-line If, got none")
	}
}

// TestSingleLineIfTagBoundaryWithElse verifies Else branches across ASP tag boundaries.
func TestSingleLineIfTagBoundaryWithElse(t *testing.T) {
	source := `<% Dim x : x = 2 : If x = 1 Then Response.Write "one" %><% Else %><% Response.Write "two" %><% End If %>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := "two"
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}

// TestSingleLineIfWithoutEndIfPreserved verifies that a standard single-line If statement without End If
// across tag boundaries continues executing subsequent statements unconditionally (IIS 10.0 behavior).
func TestSingleLineIfWithoutEndIfPreserved(t *testing.T) {
	source := `<% Dim x : x = 1 : If x = 1 Then Response.Write "one" %><% Response.Write "two" %>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := "onetwo"
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}

// TestSelectCaseTagBoundaryReproduction validates Issue 2 reproduction where Select Case is followed
// by an ASP tag boundary before the first Case.
func TestSelectCaseTagBoundaryReproduction(t *testing.T) {
	source := `<% Dim x : x = "b" : Select Case x %><% Case "a" : Response.Write "A" %><% Case "b" : Response.Write "B" %><% End Select %>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := "B"
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}

// TestSelectCaseTagBoundaryWithLiteralHTML verifies that literal HTML and whitespace between Select Case
// and the first Case statement are silently bypassed in IIS 10.0 standard compliance.
func TestSelectCaseTagBoundaryWithLiteralHTML(t *testing.T) {
	source := `<%
Dim x : x = "b"
Select Case x
%>
<b>Ignored literal HTML between Select Case and Case</b>
<%
Case "a"
    Response.Write "A"
Case "b"
    Response.Write "B"
End Select
%>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := "B"
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}

// TestSelectCaseTagBoundaryBetweenCases verifies tag boundaries and newlines between multiple Case branches.
func TestSelectCaseTagBoundaryBetweenCases(t *testing.T) {
	source := `<%
Dim x : x = "b"
Select Case x
%>
<% Case "a" %>
<% Response.Write "Alpha" %>
<% Case "b" %>
<% Response.Write "Beta" %>
<% End Select %>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := "Beta"
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}

// TestSelectCaseTagBoundaryCaseElse verifies Case Else following tag boundaries.
func TestSelectCaseTagBoundaryCaseElse(t *testing.T) {
	source := `<%
Dim x : x = "z"
Select Case x
%>
<% Case "a" : Response.Write "A" %>
<% Case Else : Response.Write "Default" %>
<% End Select %>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := "Default"
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}

// TestSelectCaseTagBoundaryEmpty verifies Select Case with no cases across tag boundaries.
func TestSelectCaseTagBoundaryEmpty(t *testing.T) {
	source := `<%
Dim x : x = 1
Select Case x
%>
<% End Select %>`
	compiler := NewASPCompiler(source)
	if err := compiler.Compile(); err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	out := runVBSAndGetOutput(t, source)
	expected := ""
	if out != expected {
		t.Fatalf("unexpected output: got %q, want %q", out, expected)
	}
}
