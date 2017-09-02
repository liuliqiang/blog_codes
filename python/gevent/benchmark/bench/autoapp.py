# -*- coding: utf-8 -*-
"""Create an application instance."""
from flask.helpers import get_debug_flag

from bench.app import create_app
from bench.settings import DevConfig, ProdConfig

CONFIG = DevConfig # if get_debug_flag() else ProdConfig

app = create_app(CONFIG)
with app.app_context():
    from bench.extensions import db
    db.create_all()
app.run(debug=True)