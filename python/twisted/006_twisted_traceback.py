#!/usr/bin/env python
# encoding: utf-8

"""
    A simple python script template.
    Created by yetship at 2017-08-09 21:35
"""
import traceback

from twisted.internet import reactor


def stack():
    print("The python stack")
    traceback.print_stack()


reactor.callWhenRunning(stack)
reactor.run()
