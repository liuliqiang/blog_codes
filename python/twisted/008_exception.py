#!/usr/bin/env python
# encoding: utf-8

"""
    A simple python script template.
    Created by yetship at 2017-08-09 21:51
"""
from twisted.internet import reactor


def fallback():
    raise Exception("I fall down")


def upagain():
    print("But i get up again")


reactor.callWhenRunning(fallback)
reactor.callWhenRunning(upagain)
print("Starting the reactor")
reactor.run()
