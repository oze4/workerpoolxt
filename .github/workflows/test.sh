#!/usr/bin/env bash

# ================================================= #
#                                                   #
# Thanks to https://github.com/gammazero/workerpool #
#                                                   #
# ================================================= #

set -e
echo "" > coverage.out

for d in $(go list ./... | grep -v vendor); do
    # specific race condition needs to be thoroughly tested to show its ugly face (hence -count=100000)
    go test -race -run ^TestStopRace$ -count=100000 $d 
    go test -race -run ^TestSubmitWithSubmitXT_UsingStopWaitXT$ -count=2750 $d
    go test -race -run ^TestSubmitWithSubmitXT_UsingStopWait$ -count=2750 $d
    # basic, defaut test cmd
    go test -race -coverprofile=profile.out -covermode=atomic $d

    if [ -f profile.out ]; then
        cat profile.out >> coverage.out
        rm profile.out
    fi
done