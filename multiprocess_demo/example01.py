#!/usr/bin/env python

"""
    A simple python example for multiprocess.
    Created by yetship at 2017/4/18 08:57
"""
from time import time
import multiprocessing


def worker():
    """Worker is process's body"""
    print(time())


if __name__ == '__main__':
    jobs = []
    for i in range(3):
        p = multiprocessing.Process(target=worker)
        jobs.append(p)
        p.start()
