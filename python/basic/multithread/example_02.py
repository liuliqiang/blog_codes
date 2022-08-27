#!/usr/bin/env python
# encoding: utf-8
import threading


def target_func():
    thread_obj = threading.current_thread()
    print("thread daemon: {}".format(thread_obj.daemon))
    print("thread name: {}".format(thread_obj.getName()))
    print("thread ident: {}".format(thread_obj.ident))
    print("thread isAlive: {}".format(thread_obj.isAlive()))
    print("thread isDaemon: {}".format(thread_obj.isDaemon()))
    print("thread is_alive: {}".format(thread_obj.is_alive()))
    print("thread name is: {}".format(thread_obj.name))

threading.Thread(name="Hehe", target=target_func).start()
