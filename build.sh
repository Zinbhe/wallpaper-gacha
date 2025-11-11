#! /bin/sh

mkdir -p ./bin

CGO_ENABLED=1 go build -v -o wallpaper-gacha

mv ./wallpaper-gacha ./bin
