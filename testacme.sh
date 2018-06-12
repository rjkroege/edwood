#!/bin/bash

mkdir /tmp/testacme
export NAMESPACE=/tmp/testacme
acme  $*

