#!/usr/bin/env bash
# Rewrite the Go module path so this fork is addressable by the Go toolchain.
#
# Upstream declares `module g3pix.com.br/axonasp/v2`. A fork cannot keep that
# path: `go get` would resolve it back to upstream, and a `replace` directive
# still requires the replacement's own go.mod to declare the path it is served
# from. So the fork renames itself, and consumers import it directly.
#
# This rename is mechanical and behaviour-free. It is kept as a script rather
# than a hand-edited diff so that rebasing onto a new upstream release stays a
# two-minute job:
#
#     git fetch upstream --tags
#     git checkout -b pgs/vX.Y.Z vX.Y.Z
#     ./rename-module-path.sh
#     git commit -am "Rename module path to github.com/peoplegroupservices/axonasp/v2"
#     ... then cherry-pick the patch commits listed in FORK.md ...
#     git tag vX.Y.Z-pgs.N && git push origin vX.Y.Z-pgs.N
#
# History: this script originally existed because upstream tagged v2.x while
# declaring a module path with no /v2 suffix, so no v2 release could be imported
# at all (guimaraeslucas/axonasp#113). That is fixed upstream; the script now
# exists only to make the fork addressable.

set -euo pipefail

OLD="g3pix.com.br/axonasp/v2"
NEW="github.com/peoplegroupservices/axonasp/v2"

cd "$(dirname "$0")"

# Import paths in Go source. Only quoted occurrences, so prose in comments that
# mentions the project's canonical name is left alone.
grep -rl "\"$OLD" --include='*.go' . | xargs sed -i "s|\"$OLD|\"$NEW|g"

# The module declaration itself.
sed -i "s|^module $OLD\$|module $NEW|" go.mod

echo "Module path is now $NEW"
echo "Remaining quoted references to $OLD in Go source: $(grep -rl "\"$OLD" --include='*.go' . | wc -l)"
