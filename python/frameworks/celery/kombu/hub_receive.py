# -*- coding: utf-8 -*-

"""
    Author: liuliqiang@smartx.com
    Created at: 2017-10-23 09:32
    Update: -

    Copyright (c) 2013-2017, SMARTX
    All rights reserved.

    Example for receive task async
"""

from __future__ import absolute_import, unicode_literals

import sys

from kombu import Connection, Exchange, Queue, Producer, Consumer
from kombu.async import Hub


def main(arguments):
    hub = Hub()
    exchange = Exchange('asynt')
    queue = Queue('asynt', exchange, 'asynt')

    def send_message(conn):
        producer = Producer(conn)
        producer.publish('hello world', exchange=exchange, routing_key='asynt')
        print('message sent')

    def on_message(message):
        print('received: {0!r}'.format(message.body))
        message.ack()
        hub.stop()  # <-- exit after one message

    conn = Connection('redis://localhost:6379')
    conn.register_with_event_loop(hub)

    with Consumer(conn, [queue], on_message=on_message):
        send_message(conn)
        hub.run_forever()


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
