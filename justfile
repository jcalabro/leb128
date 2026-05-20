default: test

test:
    go test -race -count=1 ./...

bench:
    go test -run='^$' -bench=. -benchmem -benchtime=2s -count=10 ./...

fuzz time='30s':
    #!/usr/bin/env bash
    set -euo pipefail
    for target in $(go test -list '^Fuzz' ./... | grep '^Fuzz'); do
        echo "=== $target ==="
        go test -run='^$' -fuzz="^${target}$" -fuzztime={{time}} ./...
    done
