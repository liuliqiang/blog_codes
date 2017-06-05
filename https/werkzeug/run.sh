#!/bin/bash

export FLAKS_KEY_PATH=/tmp/flask/key/
export FLASK_HOSTNAME=localhost
export FLASK_DEMO_SERVER_PORT=9527

if [ ! -d "$FLAKS_KEY_PATH" ]; then
    mkdir -p $FLAKS_KEY_PATH
    python make_key.py
fi

python server.py
