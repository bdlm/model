#!/bin/sh
set -e

go get -u github.com/golang/dep/cmd/dep
dep ensure

cd vendor/github.com/bdlm/std
git --no-pager log
cd -

rm -f coverage.txt
for dir in $(go list ./...); do
    echo "go test -timeout 20s -coverprofile=profile.out $dir"
    go test -timeout 20s -coverprofile=profile.out $dir
    exit_code=$?
    if [ "0" != "$exit_code" ]; then
        exit $exit_code
    fi
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
exit $exit_code
