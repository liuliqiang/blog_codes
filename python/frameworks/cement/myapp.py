#!/usr/bin/env python
# encoding: utf-8
from cement.core.foundation import CementApp
from cement.core import hook
from cement.utils.misc import init_defaults


# define our default configuration options
defaults = init_defaults('myapp')
defaults['myapp']['debug'] = False
defaults['myapp']['some_param'] = 'some value'


# define any hook functions here
def my_cleanup_hook(app):
    pass


# define the application class
class MyApp(CementApp):
    class Meta:
        label = 'myapp'
        config_defaults = defaults
        extensions = ['daemon', 'json', 'yaml']
        hooks = [
            ('pre_close', my_cleanup_hook),
        ]


with MyApp() as app:
    # add arguments to the parser
    app.args.add_argument('-f', '--foo', action='store', metavar='STR',
                          help='the notorious foo option')

    # log stuff
    app.log.debug("About to run my myapp application!")

    # run the application
    app.run()

    # continue with additional application logic
    if app.pargs.foo:
        app.log.info("Received option: foo => %s" % app.pargs.foo)
