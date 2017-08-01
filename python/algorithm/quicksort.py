#!/usr/bin/env python
# encoding: utf-8


def quick_sort(ary):
    if len(ary) < 2:
        return ary

    pivot = ary[0]
    left = [e for e in ary if e <= pivot]
    right = [e for e in ary if e > pivot]
    return quick_sort(left[1:]) + [pivot] + quick_sort(right)

if __name__ == "__main__":
    origin_list = [1, 3, 5, 7, 9, 2, 4, 6, 8, 0]
    print(quick_sort(origin_list))
    origin_list = [9, 8, 7, 6, 5, 4, 3, 2, 1, 0]
    print(quick_sort(origin_list))
    origin_list = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
    print(quick_sort(origin_list))
