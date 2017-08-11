#!/usr/bin/env python
# encoding: utf-8

if __name__ == "__main__":
    idx = 0
    a = list()

    while idx >= 0:
        a.append('{}'.format(idx) * 1024 * 1024 * 50)
        idx += 1
