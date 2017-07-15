#!/bin/bash

CONN=500

for (( i = 0; i < CONN; i++))
do
    ./multi_client >> /tmp/multi_client.log 2>&1 &
done

