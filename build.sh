#!/usr/bin/bash

o="scoreboard"
if [ $# -ge 1 ]; then
    o="$1"
fi

which packr2 &> /dev/null
if [ $? -ne 0 ]; then
    printf "Installing packr..\n"
    go get -u github.com/gobuffalo/packr/v2/...
fi

printf "Building...\n"
packr2
go build -o "$o" cmd/scoreboard/main.go
packr2 clean
printf "Done!\n"
