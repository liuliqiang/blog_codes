#!/usr/bin/env python
# encoding: utf-8
import time


class BackingStore:
    def __init__(self):
        self.data = []

    def write(self, datum):
        print('Started writing to backing store.')
        time.sleep(2)  # Writing to disk is slow
        self.data.append(datum)
        print('Finished writing to backing store.')

    def read(self, index):
        print('Started reading from backing store.')
        time.sleep(2)  # Reading from disk is slow
        print('Finished reading from backing store.')
        return self.data[index]


class Cache:
    def __init__(self):
        self.data = []

    def write(self, datum):
        print('Started writing to cache.')
        self.data.append(datum)
        print('Finished writing to cache.')

    def read(self, index):
        print('Started reading from backing store.')
        print('Finished reading from backing store.')
        return self.data[index]

backing_store = BackingStore()
cache = Cache()


def write_through(cache, backing_store, datum):
    cache.write(datum)
    backing_store.write(datum)
