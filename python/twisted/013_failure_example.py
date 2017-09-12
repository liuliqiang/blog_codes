#!/usr/bin/env python
# encoding: utf-8

"""
    A simple python script template.
    Created by yetship at 2017-08-10 11:21
"""
# This code illustrates a few aspects of Failures.
# Generally, Twisted makes Failures for us.

from twisted.python.failure import Failure

class RhymeSchemeViolation(Exception): pass


print 'Just making an exception:'
print

e = RhymeSchemeViolation()

failure = Failure(e)

# Note this failure doesn't include any traceback info
print failure

print
print

print 'Catching an exception:'
print

def analyze_poem(poem):
    raise RhymeSchemeViolation()

try:
    analyze_poem("""\
Roses are red.
Violets are violet.
That's why they're called Violets.
Duh.
""")
except:
    failure = Failure()


# This failure saved both the exception and the traceback
print failure
