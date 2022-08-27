#!/usr/bin/env python
# encoding: utf-8
"""
    要想等待守护进结束，可以使用 join，
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
d.join()
