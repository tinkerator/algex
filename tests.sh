#!/bin/bash
#
go test zappem.net/pub/math/algex/{factor,matrix,rotation,terms}

# Tests for the algex tool.
go build examples/algex.go
TMPDIR=$(mktemp -d)
for t in tests/*.ax ; do
    echo "testing: $t"
    output="${TMPDIR}/${t#*/}.actual"
    ./algex --file="$t" > "${output}"
    diff -u "${t}.ref" "${output}"
    x=${?}
    if [ ${x} -ne 0 ]; then
	break
    fi
done
rm -rf "${TMPDIR}"
if [ ${x} -ne 0 ]; then
    echo "FAILED"
else
    echo "PASSED"
fi
exit ${x}
