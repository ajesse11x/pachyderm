#!/bin/bash

set -e

go get -u golang.org/x/lint/golint
for file in $(find "./src" -name '*.go' | grep -v '/vendor/' | grep -v '\.pb\.go'); do
    if [[ $file == *fileset/tar* ]]; then
        continue
    fi
    golint -set_exit_status $file;
done;

files=$(gofmt -l . | grep -v vendor || true)

if [[ $files ]]; then
    echo Files not passing gofmt:
    tr ' ' '\n'  <<< $files
    exit 1
fi
