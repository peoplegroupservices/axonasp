# Why this fork exists

This is a fork of [guimaraeslucas/axonasp](https://github.com/guimaraeslucas/axonasp),
the AxonASP Classic ASP engine, maintained by G3Pix Ltda under MPL-2.0. All of the
engine is their work; please report engine bugs upstream, not here.

It exists for exactly one reason, and should be deleted once that reason goes away.

## The problem

Upstream's `go.mod` declares `module g3pix.com.br/axonasp` with no major-version
suffix, while the project is tagged `v2.x`. Go's semantic import versioning requires
a `/v2` suffix from v2 onwards, so the module proxy will not serve any v2 tag:

```console
$ go list -m -versions g3pix.com.br/axonasp
g3pix.com.br/axonasp v1.0.0 v1.2.5 v1.2.8
```

The practical effect is that Go library consumers are silently pinned to **v1.2.8**,
from before the v2 rewrite — `go get -u` reports it as up to date, so nothing signals
that 2.x exists. Neither a direct `require` of the GitHub path nor a `replace`
directive works around it; only a local filesystem `replace` does, which is not
something you want in a project's `go.mod`.

Reported upstream as [#113](https://github.com/guimaraeslucas/axonasp/issues/113).

## What we changed

Nothing but the module path — see `rename-module-path.sh`, which is the entire
difference from upstream and is the script that produced it. There are **no
behavioural changes**: upstream's test suite produces a byte-identical set of results
before and after the rename.

Deliberately *not* carried here: a candidate fix for
[#112](https://github.com/guimaraeslucas/axonasp/issues/112) (a VBScript apostrophe
comment swallowing the `%>` that ends the block). It is a one-condition change, but it
contradicts two upstream tests that assert the current behaviour, so it is the
maintainer's call to make. Keeping this fork free of behavioural divergence is what
makes it cheap to rebase and safe to retire.

## Using it

```go
require github.com/peoplegroupservices/axonasp/v2 v2.3.17-pgs.1
```

Tags are `<upstream version>-pgs.<n>`. The prerelease suffix sorts *below* the
upstream version it is built from, which is the correct signal: this should be
superseded the moment a real release is resolvable.

## Rebasing onto a new upstream release

```sh
git fetch upstream --tags
git checkout -b pgs/module-path-vX.Y.Z vX.Y.Z
./rename-module-path.sh
git commit -am "Rename module path to github.com/peoplegroupservices/axonasp/v2"
git tag vX.Y.Z-pgs.1 && git push origin vX.Y.Z-pgs.1
```

## Retiring it

When upstream #113 is fixed, change the consumer's `require` to the upstream module
path and delete this fork. No code changes will be needed on our side beyond the
import path, because we have not diverged from upstream in any other way.
