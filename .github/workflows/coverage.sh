#!/usr/bin/env bash

echo $GOPATH
go get github.com/mattn/goveralls
go get -u github.com/rakyll/gotest
$GOPATH/bin/goveralls -coverprofile=coverage.out -service=github -repotoken=$COVERALLS_TOKEN
