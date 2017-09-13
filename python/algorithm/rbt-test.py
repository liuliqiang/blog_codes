#!/usr/bin/env python
# encoding: utf-8

"""
    A simple python script template.
    Created by yetship at 2017-09-12 22:15
"""
from hashlib import md5
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


class CHNode(object):
    def __init__(self, host_name, id=None, vhost=None):
        self._id = id
        self.host_name = host_name
        if vhost:
            self._hash_host_name = "{}#{}".format(host_name, vhost)
        else:
            self._hash_host_name = host_name

    def get_id(self, ch_size):
        if self._id:
            return self._id
        return int(md5(self._hash_host_name.encode("utf-8")).hexdigest(), 16) % ch_size


class ConsistHash(object):
    def __init__(self, size=0xffff):
        self.size = size # set consistent hash circul size
        self.rbt = RBTree()  # red black tree

    def insert_host(self, host):
        host_id = host.get_id(self.size)
        self.rbt.insert(host_id, host)

    def remove_host(self, host):
        host_id = host.get_id(self.size)
        self.rbt.remove(host_id)

    @staticmethod
    def _find_upper(root, elem):
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

    def find_host(self, id):
        id %= self.size
        idx = self._find_upper(self.rbt._root, id)
        if idx == -1:  # id larger than max id
            # assert tree is not empty
            return self.rbt.min_item()[1]
        return self.rbt.get_value(idx)


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
    for i in range(4):
        hostname = "host{}".format(i)
        print("{}: {}".format(hostname, CHNode(hostname).get_id(0xffff)))

    ch = ConsistHash()
    ch.insert_host(CHNode("host0"))
    ch.insert_host(CHNode("host1"))
    ch.insert_host(CHNode("host2"))
    ch.insert_host(CHNode("host3"))
    search_elems = [0, 10000, 20000, 30000, 40000, 50000, 60000, 70000, 0xffff]
    for search_elem in search_elems:
        print("{} in {}".format(search_elem, ch.find_host(search_elem).host_name))
