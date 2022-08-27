#!/usr/bin/env python
# encoding: utf-8
import threading


def my_func():
    thread_obj = threading.current_thread()
    print("thread: {} runing".format(thread_obj.name))


t = threading.Thread(target=my_func)
t.start()

print("all thread done!")
