#!/usr/bin/env python
# encoding: utf-8

"""
    A simple python script template.
    Created by yetship at 2017-08-09 21:35
"""
from twisted.internet import reactor


def hello():
    print("Hello from the reactor loop")
    print("Lately i fell like i'm stuck in a rut")

reactor.callWhenRunning(hello)
print("Starting the reactor")
reactor.run()
