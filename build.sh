#!/usr/bin/bash

output="../bin/scoreboard"
if [ $# -ge 1 ]; then
    output="$1"
fi

which packr2 &> /dev/null
if [ $? -ne 0 ]; then
    printf "Installing packr..\n"
    bash -c "cd scoreboard; go get -u github.com/gobuffalo/packr/v2/packr2"
fi

printf "Building...\n"
bash -c "cd scoreboard; packr2; go build -trimpath -ldflags \"-s -w\" -o \"$output\" cmd/scoreboard/main.go; packr2 clean"

which upx &> /dev/null
if [ $? -eq 0 ] && [ -f "$output" ]; then
    upx --compress-exports=1 --strip-relocs=1 --compress-icons=2 --best --no-backup -9 "$output"
fi

printf "Done!\n"
