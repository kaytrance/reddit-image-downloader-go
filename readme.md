# UNIXP*ORN reddit image downloader

This small tool downloads all good looking images from [/r/unixporn](https://www.reddit.com/r/unixporn) reddit hub. It downloads only PNG quality images with only whitelisted tags.

It also keeps track of when images were downloaded so upon next launch it will get only images you don't have. Feel free to add this then to crontab to get a daily portion of awesome looking rice!


## Build instructions

run `go build -o unixpdownloader`.