# -*- coding: utf-8 -*-
import dis
import sys

if __name__ == "__main__":
    filename = sys.argv[1]
    with open(filename) as f:
        code = f.read()
    co = compile(code, filename, "exec")

    print("consts: ")
    for const in co.co_consts:
        print("\t{}".format(const))

    print("names: ")
    for name in co.co_names:
        print("\t{}".format(name))

    print("freevars: ")
    for freevar in co.co_freevars:
        print("\t{}".format(freevar))

    print("codes: ")
    dis.dis(co.co_code)

    print("\nnlocols: {}\n".format(co.co_nlocals))

    print("stacksize: {}\n".format(co.co_stacksize))
