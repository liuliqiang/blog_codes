#!/usr/bin/env python
# encoding: utf-8

"""
    A simple python script template.
    Created by yetship at 2017-09-12 22:15
"""
from bintrees.rbtree import RBTree


def _not_exists(key):
    return key is None or key == -1


# 找上限
def find_upper(root, elem):
    if root is None:
        return -1

    if elem == root.key:
        return root.key
    elif elem < root.key:
        maybe_max = find_upper(root.left, elem)
        if _not_exists(maybe_max):
            return root.key
        return maybe_max
    else:
        maybe_max = find_upper(root.right, elem)
        if _not_exists(maybe_max):
            return -1
        return maybe_max


# 找下限
def find_lower(root, elem):
    if root is None:
        return -1

    if elem == root.key:
        return root.key
    elif elem < root.key:
        maybe_min = find_lower(root.left, elem)
        if _not_exists(maybe_min):
            return -1
        return maybe_min
    else:
        maybe_min = find_lower(root.right, elem)
        if _not_exists(maybe_min):
            return root.key
        return maybe_min


def find_next(root, elem):
    if root is None:
        return None, None
    return find_lower(root, elem), find_upper(root, elem)


def trace_tree(root):
    if root:
        print("{} {} {}".format(
            root.key,
            root.left.key if root.left is not None else None,
            root.right.key if root.right is not None else None))
        trace_tree(root.left)
        trace_tree(root.right)


if __name__ == "__main__":
    rbt = RBTree()
    for i in range(0, 30, 2):
        rbt.insert(i, "{}".format(i))
    print(find_next(rbt._root, 3))
    print(find_next(rbt._root, 4))
    print(find_next(rbt._root, 5))
    print(find_next(rbt._root, 21))
    print(find_lower(rbt._root, 10)) # should be 10
    print(find_lower(rbt._root, 11)) # should be 10
    print(find_lower(rbt._root, 9)) # should be 8
    print(find_lower(rbt._root, 7)) # should be 6
    print(find_lower(rbt._root, 5)) # should be 4
    print(find_lower(rbt._root, 3)) # should be 2
    print(find_lower(rbt._root, 1)) # should be 0
    print(find_lower(rbt._root, 0)) # should be 0
    print(find_lower(rbt._root, -2)) # should be -1
    print(find_upper(rbt._root, 10)) # should be 10
    print(find_upper(rbt._root, 11)) # should be 12
    print(find_upper(rbt._root, 9)) # should be 10
    print(find_upper(rbt._root, 7)) # should be 8
    print(find_upper(rbt._root, 5)) # should be 6
    print(find_upper(rbt._root, 3)) # should be 4
    print(find_upper(rbt._root, 1)) # should be 2
    print(find_upper(rbt._root, 0)) # should be 0
    print(find_upper(rbt._root, 31)) # should be -1
    # trace_tree(rbt._root)
