# -*- coding: utf-8 -*-
# Copyright (c) 2013-2017, SMARTX
# All rights reserved.

"""
    Comment for this files.
    Author: liuliqiang@smartx.com
    Created at: 2017-11-14 09:39
    Update: -
"""

import sys

import kombu
from kombu import pidbox


def callback():
    print("callback")


def main(arguments):
    connection = kombu.Connection("redis://192.168.25.2:6379")
    mailbox = pidbox.Mailbox("test", type="direct")
    bound = mailbox(connection)

    bound._broadcast("print_msg", {'msg': 'Message for you'})
    # info = bound.(["test_application"],"list", callback=callback)

if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
