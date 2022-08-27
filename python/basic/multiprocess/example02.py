#!/usr/bin/env python

"""
    A simple python example for multiprocess.
    Created by yetship at 2017/4/18 08:57
"""
from random import randint
import multiprocessing


def worker(lower, upper):
    """thread worker function"""
    print("get a random int: {}".format(randint(lower, upper)))


if __name__ == '__main__':
    for i in range(3):
        p = multiprocessing.Process(target=worker, args=(i, i ** 2))
        p.start()