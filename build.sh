#!/usr/bin/bash

OUT="scoreboard"
if [ $# -ge 1 ]; then
    OUT="$1"
fi

echo "Building..."
packr2
go build -o "$OUT" cmd/scoreboard/main.go
packr2 clean
echo "Done!"
