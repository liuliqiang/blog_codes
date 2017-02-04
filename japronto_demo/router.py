#!/usr/bin/env python
# encoding: utf-8
from japronto import Application

app = Application()
r = app.router


def slash(req):
    return req.Response(text='Hello {}'.format(req.method))
r.add_route('/', slash)


def get_love(req):
    return req.Response(text='Got some love')
r.add_route('/love', get_love, 'GET')


def methods(req):
    return req.Response(text=req.method)
r.add_route('/methods', methods, methods=['POST', 'DELETE'])


def params(req):
    return req.Response(text=str(req.match_dict))
r.add_route('/params/{p1}/{p2}', params)


app.run(debug=True)
