# -*- coding: utf-8 -*-

"""
    Author: liuliqiang@smartx.com
    Created at: 2017-10-23 09:28
    Update: -

    Copyright (c) 2013-2017, SMARTX
    All rights reserved.

    Example of simple consumer that waits for a single message, acknowledges it
    and exits.
"""

from __future__ import absolute_import, unicode_literals, print_function

import sys
from pprint import pformat

from kombu import Connection, Exchange, Queue, Consumer, eventloop


def main(arguments):
    #: By default messages sent to exchanges are persistent (delivery_mode=2),
    #: and queues and exchanges are durable.
    exchange = Exchange('kombu_demo', type='direct')
    queue = Queue('kombu_demo', exchange, routing_key='kombu_demo')

    def pretty(obj):
        return pformat(obj, indent=4)

    #: This is the callback applied when a message is received.
    def handle_message(body, message):
        print('Received message: {0!r}'.format(body))
        print('  properties:\n{0}'.format(pretty(message.properties)))
        print('  delivery_info:\n{0}'.format(pretty(message.delivery_info)))
        message.ack()

    #: Create a connection and a channel.
    #: If hostname, userid, password and virtual_host is not specified
    #: the values below are the default, but listed here so it can
    #: be easily changed.
    with Connection('redis://localhost:6379') as connection:

        #: Create consumer using our callback and queue.
        #: Second argument can also be a list to consume from
        #: any number of queues.
        with Consumer(connection, queue, callbacks=[handle_message]):

            #: Each iteration waits for a single event.  Note that this
            #: event may not be a message, or a message that is to be
            #: delivered to the consumers channel, but any event received
            #: on the connection.
            for _ in eventloop(connection):
                pass


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
