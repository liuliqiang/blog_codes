#!/usr/bin/env python
# encoding: utf-8
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

threading.Thread(name="daemon", target=daemon, daemon=True).start()
threading.Thread(name="nondaemon", target=non_daemon).start()
