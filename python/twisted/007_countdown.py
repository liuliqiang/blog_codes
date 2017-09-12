#!/usr/bin/env python
# encoding: utf-8

"""
    A simple python script template.
    Created by yetship at 2017-08-09 21:46
"""
from twisted.internet import reactor


class Countdown(object):
    counter = 5

    def count(self):
        if self.counter == 0:
            reactor.stop()
        else:
            print(self.counter, "...")
            self.counter -= 1
            reactor.callLater(1, self.count)

reactor.callWhenRunning(Countdown().count)
print("Start")
reactor.run()
print("Stop!")
