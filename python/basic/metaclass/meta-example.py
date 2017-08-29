#!/usr/bin/env python
# encoding: utf-8
from warnings import warn


class MetaC(type):
    def __init__(cls, name, bases, attrd):
        super(MetaC, cls).__init__(name, bases, attrd)

        if '__str__' not in attrd:
            raise TypeError("Class requires overriding of __str__()")

        if '__repr__' not in attrd:
            warn("Class suggests overriding of __repr__()", stacklevel=3)


class Foo(object):
    __metaclass__ = MetaC
