#!/usr/bin/env python
# encoding: utf-8
from japronto import Application

def hello(req):
    return req.Response("hello world")


app = Application()
app.router.add_route('/', hello)
app.run(debug=True)
