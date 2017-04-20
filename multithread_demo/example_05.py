#!/usr/bin/env python
# encoding: utf-8
"""
    join 也是可以有时间限制的
"""
import threading
import time
import logging


logging.basicConfig(
    level=logging.DEBUG,
    format="[%(levelname)s] (%(threadName)-18s) %(message)s"
)


def daemon():
    logging.debug("Deamon Starting")
    time.sleep(1)
    logging.debug("Daemon Exiting")


def non_daemon():
    logging.debug("NonDaemon Starting")
    logging.debug("NOnDaemon Exiting")

d = threading.Thread(name="daemon", target=daemon, daemon=True)
t = threading.Thread(name="nondaemon", target=non_daemon)

d.start()
t.start()
d.join(0.1)
logging.debug("thread d isAlive:{}".format(d.isAlive()))
