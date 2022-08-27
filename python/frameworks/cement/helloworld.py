#!/usr/bin/env python
# encoding: utf-8
from cement.core.foundation import CementApp

with CementApp('helloworld') as app:
    app.run()
    print('Hello World')
