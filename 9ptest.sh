#!/bin/sh
 rm /tmp/ns.test_acme/acme
 ./edwood &
sleep 2
 echo this is a test  |9p write acme/new/body
 9p read acme/2/body
 echo '#5' | 9p write acme/2/addr
 echo 'dot=addr' | 9p write acme/2/ctl
