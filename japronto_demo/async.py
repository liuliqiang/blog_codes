#!/usr/bin/env python
# encoding: utf-8
import asyncio
from japronto import Application


def synchronous(req):
    return req.Response(text='I am synchronous')


async def asynchronous(req):
    for i in range(1, 4):
        await asyncio.sleep(1)
        print(i, 'seconds elapsed')

    return req.Response(text='3 senconds elapsed')


app = Application()
app.router.add_route('/sync', synchronous)
app.router.add_route('/async', asynchronous)
app.run(debug=True)
