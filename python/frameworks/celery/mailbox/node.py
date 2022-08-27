# -*- coding: utf-8 -*-
# Copyright (c) 2013-2017, SMARTX
# All rights reserved.

"""
    Comment for this files.
    Author: liuliqiang@smartx.com
    Created at: 2017-11-14 09:34
    Update: -
"""
import sys

import kombu
from kombu import pidbox

hostname = "localhost"

connection = kombu.Connection("redis://192.168.25.2:6379")
mailbox = pidbox.Mailbox("test", type="direct")
node = mailbox.Node(hostname, state={"a": "b"})
node.channel = connection.channel()


def callback(body, message):
    print(body)
    print(message)


def main(arguments):
    consumer = node.listen(callback=callback)
    try:
        while True:
            print('Consumer Waiting')
            connection.drain_events()
    finally:
        consumer.cancel()

    def beat(x):
        print(x)


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
