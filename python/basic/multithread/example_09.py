#!/usr/bin/env python
# encoding: utf-8

from random import randint
from time import sleep
from queue import Queue
from threading import Thread


def write_queue(queue):
    print('producing object for Q')
    queue.put('xxx', 1)
    print("size now: {}".format(queue.qsize()))


def read_queue(queue):
    queue.get(1)
    print("consumed object from Q, queue size: {}".format(queue.qsize()))


def writer(queue):
    n = randint(2, 4)
    for i in range(n):
        write_queue(queue)
        sleep(randint(1, 3))


def reader(queue):
    n = randint(2, 4)
    for i in range(n):
        read_queue(queue)
        sleep(randint(2, 5))


funcs = [writer, reader]


def main():
    q = Queue(32)

    threads = []
    for f in funcs:
        t = Thread(target=f, args=(q, ), name=f.__name__)
        threads.append(t)

    for t in threads:
        t.start()

    for t in threads:
        t.join()

    print("all done!")


if __name__ == "__main__":
    main()
