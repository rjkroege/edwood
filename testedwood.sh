#!/bin/bash

mkdir /tmp/testedwood
export NAMESPACE=/tmp/testedwood
./edwood -validateboxes $*

