#!/bin/bash

mkdir -p /tmp/testacme
export NAMESPACE=/tmp/testacme
acme  $*

