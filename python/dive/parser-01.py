# -*- coding: utf-8 -*-

filename = "simple_obj.py"
with open(filename) as f:
    code = f.read()
co = compile(code, filename, "exec")

print("consts: ")
for const in co.co_consts:
    print("\t{}".format(const))

print("names: ")
for name in co.co_names:
    print("\t{}".format(name))
