#!/usr/bin/env python
# encoding: utf-8
import time
import random
import logging
import threading


def worker():
    pause = random.randint(1, 5)/ 10
    logging.debug('sleep: {}'.format(pause))
    time.sleep(pause)
    logging.debug('ending')


logging.basicConfig(
    level=logging.DEBUG,
    format="[%(levelname)s] (%(threadName)-18s) %(message)s"
)

for i in range(3):
    t = threading.Thread(target=worker, daemon=True)
    t.start()

main_thread = threading.main_thread()
for t in threading.enumerate():
    if t is main_thread:
        continue
    logging.debug("joining: {}".format(t.name))
    t.join()
