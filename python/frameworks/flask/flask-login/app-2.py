#!/usr/bin/env python
# encoding: utf-8
from flask import Flask, Blueprint
from flask.ext.login import LoginManager, login_required

app = Flask(__name__)

# 以下这段是新增加的============
app.secret_key = 's3cr3t'
login_manager = LoginManager()
login_manager.session_protection = 'strong'
login_manager.login_view = 'auth.login'
login_manager.init_app(app)

@login_manager.user_loader
def load_user(user_id):
    return None
# 以上这段是新增加的============

auth = Blueprint('auth', __name__)

@auth.route('/login', methods=['GET', 'POST'])
def login():
    return "login page"

@auth.route('/logout', methods=['GET', 'POST'])
@login_required
def logout():
    return "logout page"

# test method
@app.route('/test')
@login_required
def test():
    return "yes , you are allowed"

app.register_blueprint(auth, url_prefix='/auth')
app.run(debug=True)
