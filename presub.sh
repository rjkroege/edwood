#!/bin/bash

diff -u <(echo -n) <(gofmt -d -s .) && \
go vet . && \
misspell -error . && \
staticcheck -checks inherit,-U1000,-SA4003 ./...
