#!/bin/sh
set -e

echo "go get -u github.com/golang/dep/cmd/dep"
go get -u github.com/golang/dep/cmd/dep
echo "dep ensure"
dep ensure

echo "cd vendor/github.com/bdlm/std"
cd vendor/github.com/bdlm/std
echo "git --no-pager log"
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
