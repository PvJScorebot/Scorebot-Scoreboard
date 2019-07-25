#!/usr/bin/bash

echo "Building..."
packr2
go build -o scorebard cmd/scoreboard/main.go
packr2 clean
echo "Done!"
