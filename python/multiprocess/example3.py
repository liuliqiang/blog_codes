#!/usr/bin/env python
# encoding: utf-8
import os
import time
import signal
from multiprocessing import Process, cpu_count, Queue, JoinableQueue
import multiprocessing as mp


def handler(signum, frame):
    print("Signal handler called with signal: {}".format(signum))

signal.signal(signal.SIGTERM, handler)

def worker():
    print(__name__)
    print("I am worker at processor")


def main():
    for i in range(cpu_count()):
        p = Process(target=worker)
        p.start()
        p.join()


if __name__ == "__main__":
    mp.set_start_method("forkserver")
    main()
