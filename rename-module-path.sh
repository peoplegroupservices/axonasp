#!/usr/bin/env bash
# Rewrite the Go module path so this fork is resolvable by the Go toolchain.
#
# Upstream declares `module g3pix.com.br/axonasp` while tagging v2.x releases.
# Go's semantic import versioning requires a /v2 suffix from v2 onwards, so the
# module proxy serves nothing above v1.2.8 and no v2 release can be imported as a
# library at all. Reported upstream as:
#
#     https://github.com/guimaraeslucas/axonasp/issues/113
#
# This script is the *entire* difference between this fork and upstream. Keeping it
# as a script rather than a hand-edited diff means rebasing onto a new upstream
# release is mechanical:
#
#     git fetch upstream --tags
#     git checkout -b pgs/module-path-vX.Y.Z vX.Y.Z
#     ./rename-module-path.sh
#     git commit -am "Rename module path to github.com/peoplegroupservices/axonasp/v2"
#     git tag vX.Y.Z-pgs.1 && git push origin vX.Y.Z-pgs.1
#
# Retire the fork entirely once upstream #113 is fixed.

set -euo pipefail

OLD="g3pix.com.br/axonasp"
NEW="github.com/peoplegroupservices/axonasp/v2"

cd "$(dirname "$0")"

# Import paths in Go source. Only quoted occurrences, so prose in comments that
# mentions the project's canonical name is left alone.
grep -rl "\"$OLD" --include='*.go' . | xargs sed -i "s|\"$OLD|\"$NEW|g"

# The module declaration itself.
sed -i "s|^module $OLD\$|module $NEW|" go.mod

echo "Module path is now $NEW"
echo "Remaining quoted references to $OLD in Go source: $(grep -rc "\"$OLD" --include='*.go' . | grep -v ':0$' | wc -l)"
