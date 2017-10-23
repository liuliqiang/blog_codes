# -*- coding: utf-8 -*-

"""
    Author: liuliqiang@smartx.com
    Created at: 2017-10-23 09:15
    Update: -

    Copyright (c) 2013-2017, SMARTX
    All rights reserved.

Example producer that sends a single message and exits.

You can use `complete_receive.py` to receive the message sent.

"""
from __future__ import absolute_import, unicode_literals

import sys
from kombu import Connection, Producer, Exchange


def main(arguments):
    #: By default messages sent to exchanges are persistent (delivery_mode=2),
    #: and queues and exchanges are durable.
    exchange = Exchange('kombu_demo', type='direct')
    # queue = Queue('kombu_demo', exchange, routing_key='kombu_demo')

    with Connection('redis://localhost:6379') as connection:
        #: Producers are used to publish messages.
        #: a default exchange and routing key can also be specified
        #: as arguments the Producer, but we rather specify this explicitly
        #: at the publish call.
        producer = Producer(connection)

        #: Publish the message using the json serializer (which is the default),
        #: and zlib compression.  The kombu consumer will automatically detect
        #: encoding, serialization and compression used and decode accordingly.
        producer.publish({'hello': 'world'},
                         exchange=exchange,
                         routing_key='kombu_demo',
                         serializer='json', compression='zlib')


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
