#!/usr/bin/env python
# encoding: utf-8
from flask import Flask, Blueprint
from flask.ext.login import (LoginManager, login_required, login_user,
                                logout_user, UserMixin)

app = Flask(__name__)


# user models
class User(UserMixin):
    def is_authenticated(self):
        return True

    def is_actice(self):
        return True

    def is_anonymous(self):
        return False

    def get_id(self):
        return "1"

# flask-login
app.secret_key = 's3cr3t'
login_manager = LoginManager()
login_manager.session_protection = 'strong'
login_manager.login_view = 'auth.login'
login_manager.init_app(app)

@login_manager.user_loader
def load_user(user_id):
    user = User()
    return user

auth = Blueprint('auth', __name__)

@auth.route('/login', methods=['GET', 'POST'])
def login():
    user = User()
    login_user(user)
    return "login page"

@auth.route('/logout', methods=['GET', 'POST'])
@login_required
def logout():
    logout_user()
    return "logout page"

# test method
@app.route('/test')
@login_required
def test():
    return "yes , you are allowed"

app.register_blueprint(auth, url_prefix='/auth')
app.run(debug=True)
