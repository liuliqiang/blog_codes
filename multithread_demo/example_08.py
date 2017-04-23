#!/usr/bin/env python
# encoding: utf-8
import time
import threading


def my_func():
    thread_obj = threading.current_thread()
    time.sleep(1)
    print("thread: {} runing".format(thread_obj.name))


t = threading.Thread(target=my_func, daemon=True)
t.start()
t.join()

print("all thread done!")
