#!/bin/bash
#
go test zappem.net/pub/math/algex/{factor,matrix,rotation,terms}
if [ ${?} -ne 0 ]; then
    echo "FAILED"
    exit 1
fi

# Tests for the algex tool.
go build examples/algex.go
if [ ${?} -ne 0 ]; then
    echo "FAILED"
    exit 1
fi
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
