#!/usr/bin/env python
# encoding: utf-8
from flask import Flask, Blueprint

app = Flask(__name__)

# url redirect
auth = Blueprint('auth', __name__)


@auth.route('/login', methods=['GET', 'POST'])
def login():
    return "login page"


@auth.route('/logout', methods=['GET', 'POST'])
def logout():
    return "logout page"


# test method
@app.route('/test')
def test():
    return "yes , you are allowed"

app.register_blueprint(auth, url_prefix='/auth')
app.run(debug=True)
