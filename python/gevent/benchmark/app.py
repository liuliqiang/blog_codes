#!/usr/bin/env python
# encoding: utf-8
from datetime import datetime

from flask import Flask, jsonify
from flask_login import UserMixin
from flask_sqlalchemy import SQLAlchemy

from gevent.wsgi import WSGIServer


app = Flask(__name__)
app.config.update({"SQLALCHEMY_DATABASE_URI": 'mysql+pymysql://yetship:password@lenove/test?charset=utf8&use_unicode=1',
                   "SQLALCHEMY_TRACK_MODIFICATIONS": False})

db = SQLAlchemy(app)
Column = db.Column


class User(db.Model, UserMixin):
    __tablename__ = 'users'

    user_id = db.Column(db.Integer, primary_key=True)
    username = Column(db.String(80), nullable=False)
    email = Column(db.String(80), nullable=False)
    #: The hashed password
    password = Column(db.Binary(128), nullable=True)
    created_at = Column(db.DateTime, nullable=False, default=datetime.utcnow)
    first_name = Column(db.String(30), nullable=True)
    last_name = Column(db.String(30), nullable=True)
    active = Column(db.Boolean(), default=False)
    is_admin = Column(db.Boolean(), default=False)


@app.route("/")
def index():
    u = User.query.first()
    return jsonify({"count": u.username})


if __name__ == "__main__":
    with app.app_context():
        db.create_all()
    app.run(debug=False, threaded=True, processes=1)
    # http_server = WSGIServer(('', 5000), app)
    # http_server.serve_forever()
