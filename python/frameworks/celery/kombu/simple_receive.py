# -*- coding: utf-8 -*-

"""
    Author: liuliqiang@smartx.com
    Created at: 2017-10-23 09:20
    Update: -

    Copyright (c) 2013-2017, SMARTX
    All rights reserved.

    Example receiving a message using the SimpleQueue interface.
"""

from __future__ import absolute_import, unicode_literals

import sys

from kombu import Connection


def main(arguments):
    #: Create connection
    #: If hostname, userid, password and virtual_host is not specified
    #: the values below are the default, but listed here so it can
    #: be easily changed.
    with Connection('redis://localhost:6379') as conn:

        #: SimpleQueue mimics the interface of the Python Queue module.
        #: First argument can either be a queue name or a kombu.Queue object.
        #: If a name, then the queue will be declared with the name as the queue
        #: name, exchange name and routing key.
        with conn.SimpleQueue('kombu_demo') as queue:
            message = queue.get(block=True, timeout=10)
            message.ack()
            print(message.payload)

    ####
    #: If you don't use the with statement then you must aways
    # remember to close objects after use:
    #   queue.close()
    #   connection.close()


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
