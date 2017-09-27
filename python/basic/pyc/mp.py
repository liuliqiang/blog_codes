#!/usr/bin/env python
# encoding: utf-8
import os
from multiprocessing import Pool


def resatrt_rest():
    os.system("systemctl restart zbs-rest-server")


if __name__ == '__main__':
    while True:
        p = Pool(processes=4)
        p.apply_async(resatrt_rest)
