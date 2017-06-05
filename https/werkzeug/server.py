#!/usr/bin/env python
# encoding: utf-8
import os

from werkzeug.serving import run_simple
from werkzeug.wrappers import Request, Response
from werkzeug.wsgi import SharedDataMiddleware


class HttpsServer(object):
    def dispatch_request(self, request):
        return Response('Hello Werkzeug!')

    def wsgi_app(self, environ, start_response):
        request = Request(environ)
        response = self.dispatch_request(request)
        return response(environ, start_response)

    def __call__(self, environ, start_response):
        return self.wsgi_app(environ, start_response)


def create_app(with_static=True):
    app = HttpsServer()
    if with_static:
        app.wsgi_app = SharedDataMiddleware(app.wsgi_app, {
            '/static': os.path.join(os.path.dirname(__file__), 'static')
        })
    return app


if __name__ == "__main__":
    key_path = os.environ.get("FLAKS_KEY_PATH")
    host = os.environ.get("FLASK_HOSTNAME")
    port = int(os.environ.get("FLASK_DEMO_SERVER_PORT"))

    ctx = (os.path.join(key_path, 'ssl.crt'),
           os.path.join(key_path, 'ssl.key'))

    app = create_app()
    run_simple(host, port, app, use_debugger=True, use_reloader=True,
               ssl_context=ctx)
