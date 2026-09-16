# Why this fork exists

This is a fork of [AxonASP](https://github.com/guimaraeslucas/axonasp), the
cross-platform Classic ASP engine, maintained for the PGS legacy portal.

It exists for one reason: **cadence**. We work on the portal on Wednesdays, so a
one-working-day turnaround upstream costs us a week of calendar, and a bug that
needs two round trips costs a month. Upstream is responsive and has taken our
reports nearly verbatim — this is not a quality disagreement. We simply cannot
block a Wednesday on a fix landing somewhere else.

## How it is run

**A patch queue, not a project.** Every patch here is expected to die. The rules:

1. Base on an upstream release, never on our own history.
2. One patch per commit, each with a test that pins the behaviour.
3. Every patch is also filed upstream. Drop ours when theirs lands.
4. Never refactor, never restructure, never "improve" anything. A patch earns
   its place by fixing either a construct the portal actually uses, or a
   pathology that blocks our own workflow. Nothing else.

Rule 4 is the one that matters. The failure mode is this fork quietly becoming
something we maintain, at which point we own an ASP engine forever. The second
half of it is deliberately narrow: patch 2 below is a performance fix, which is
not a correctness bug in any construct, but an hour of CI per commit is a direct
tax on a team that works one day a week.

## Rebasing onto a new upstream release

```sh
git fetch upstream --tags
git checkout -b pgs/vX.Y.Z vX.Y.Z
./rename-module-path.sh
git commit -am "Rename module path to github.com/peoplegroupservices/axonasp/v2"
# then cherry-pick each patch commit listed below that upstream has not yet fixed
git tag vX.Y.Z-pgs.N && git push origin vX.Y.Z-pgs.N
```

`rename-module-path.sh` is mechanical and behaviour-free. A fork cannot keep
upstream's `g3pix.com.br/axonasp/v2` path: `go get` would resolve it back to
upstream, and a `replace` directive does not help, because the replacement's own
go.mod still has to declare the path it is served from.

## Current patches

### 1. `If cond Then <stmt> %>` must not absorb a following `End If`

Upstream: [#129](https://github.com/guimaraeslucas/axonasp/issues/129), and
[#124](https://github.com/guimaraeslucas/axonasp/issues/124) for the second half.

VBScript's rule, measured on IIS 10.0 / VBScript 5.8.16384 on 2026-09-16:

| construct | IIS |
| --- | --- |
| `If c Then <stmt> %>` … `<% End If` | **rejected (HTTP 500)** |
| `ElseIf c Then <stmt> %>` … `<% End If` | **accepted** |

A fresh `If cond Then <stmt>` is a *complete single-line If* — the tag boundary
ends it, so a following `End If` belongs to an enclosing block, or to nothing at
all. An `ElseIf` with a statement sits inside an already-open block, so the
block stays open and `End If` closes it.

Upstream v2.3.20 changed the first form (v2.3.19 had it right) and left the
second — the one #124 reported — still failing. This patch restores the first
and fixes the second.

It breaks two upstream tests in `axonvm/single_line_if_colon_test.go`, which
assert the opposite of what IIS does; their expectations are inverted here with
the measurement cited inline. That file is where the disagreement lives if
upstream resolves #129 the other way.

`axonvm/singleline_if_tag_boundary_test.go` holds the fork's own coverage. Every
expectation in it is a measured IIS result, not a derivation.

Measure new constructs with `tools/iisprobe/singleline-if-probe.py` in the
portal repo — one construct per page, because a compile error takes down a whole
ASP page. Not yet measured, and so not asserted anywhere: real HTML between an
inline `ElseIf` arm and its `End If`, and the `Else`-across-a-boundary form that
upstream's `TestSingleLineIfTagBoundaryWithElse` covers (that test already fails
on upstream v2.3.22, before this patch).

### 2. `optimizeDeadConditionalJumpPass` was quadratic in bytecode length

Upstream: [#130](https://github.com/guimaraeslucas/axonasp/issues/130).

`findPreviousInstructionStart` rescans the bytecode from offset zero to find the
instruction before a given offset. The dead-conditional-jump pass called it once
per conditional jump while already walking the bytecode linearly, so compilation
was O(n^2) in bytecode length.

It dominated everything. On a 48k-line source, 95% of compile time was in that
one function. `Connections/ecommdata.asp` - which ~300 portal pages include -
took 9.5s to compile on its own, and the portal's full compile check took 4m37s
of wall time and 64 minutes of CPU.

The pass now records instruction starts as it walks, so the preceding
instruction is the last entry rather than a rescan.

| | before | after |
| --- | --- | --- |
| `Connections/ecommdata.asp` | 9.50s | 0.36s |
| `default.asp` | 10.49s | 0.38s |
| full portal sweep, wall | 4m37s | 12.2s |
| full portal sweep, CPU | 64m | 3m57s |

`axonvm/optimizer_dead_jump_linear_test.go` keeps the original rescanning
implementation as a test oracle and compares WHOLE COMPILES - bytecode and
program output - between the two. Comparing a single pass proves nothing,
because by the time `Compile` returns the dead jumps are already gone.

`deadConditionalJumpPass` in `compiler_optimizer.go` is a var purely so that
test can swap the implementation. It is not a configuration point.

## Upstream test suite

21 tests in `./axonvm` fail on upstream v2.3.22 on Linux before any of our
changes — filesystem, Request-collection and JScript ActiveX cases that look
environment-dependent. `TestJScriptActiveXObjectBinding` is additionally
order-dependent: it passes in isolation and fails in a full run. Compare failure
*sets* before and after a patch rather than counting, or you will chase ghosts.
