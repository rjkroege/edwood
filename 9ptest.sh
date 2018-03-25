#!/bin/bash
 rm /tmp/ns.test_acme/acme
 ./edwood &
sleep 2
 echo this is a test  |9p write acme/new/body
 9p read acme/2/body
 echo -n '1' | 9p write acme/2/addr
9p read acme/2/addr
