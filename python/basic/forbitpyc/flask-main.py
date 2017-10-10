#!/usr/bin/env python
# encoding: utf-8
from flask import Flask

from a import A


app = Flask(__name__)


@app.route('/')
def index():
    return A().a()


if __name__ == "__main__":
    app.run()
