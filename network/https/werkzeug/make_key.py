#!/usr/bin/env python
# encoding: utf-8
import os

from werkzeug.serving import make_ssl_devcert


def create_crt(path, host):
    make_ssl_devcert(path, host=host)


if __name__ == "__main__":
    path = os.environ.get("FLAKS_KEY_PATH", '/tmp/flask/key/')
    host = os.environ.get("FLASK_HOSTNAME", 'localhost')

    if not os.path.exists(os.path.join(path, "ssl.key")):
        file_prefix = os.path.join(path, "ssl")
        create_crt(file_prefix, host)
    else:
        print("ssl keys already exists")
