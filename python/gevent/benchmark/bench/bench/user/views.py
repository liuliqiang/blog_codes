# -*- coding: utf-8 -*-
"""User views."""
from flask import Blueprint, render_template, request, jsonify
from flask_login import login_required

from user.models import User

blueprint = Blueprint('user', __name__, url_prefix='/users', static_folder='../static')


@blueprint.route('/')
@login_required
def members():
    """List members."""
    return render_template('users/members.html')


@blueprint.route('/detail')
def user_detail():
    username = request.args.get('username')
    # user = db.session.query(User).filter_by(name=username).first()
    user = User.query.filter_by(username=username).first()
    if user:
        rst = {"username": username,
               "email": user.email}
    else:
        rst = {"rst": "Not Found"}
    return jsonify(**rst)
