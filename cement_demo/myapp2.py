#!/usr/bin/env python
# encoding: utf-8
from cement.core.foundation import CementApp
from cement.core.controller import CementBaseController, expose


class MyBaseController(CementBaseController):
    class Meta:
        label = 'base'
        description = "My Application does amazing things!"
        arguments = [(['-f', '--foo'], dict(action='store', help='the notorious foo option')),
                     (['-C'], dict(action='store_true', help='the big C option'))]

    @expose(hide=True)
    def default(self):
        self.app.log.info('Inside MyBaseController.default()')
        if self.app.pargs.foo:
            print("Recieved option: foo => %s" % self.app.pargs.foo)

    @expose(help="this command does relatively nothing useful")
    def command1(self):
        self.app.log.info("Inside MyBaseController.command1()")

    @expose(aliases=['cmd2'], help="more of nothing")
    def command2(self):
        self.app.log.info("Inside MyBaseController.command2()")


class MySecondController(CementBaseController):
    class Meta:
        label = 'second'
        stacked_on = 'base'

    @expose(help='this is some command', aliases=['some-cmd'])
    def second_cmd1(self):
        self.app.log.info("Inside MySecondController.second_cmd1")


class MyApp(CementApp):
    class Meta:
        label = 'myapp'
        base_controller = 'base'
        handlers = [MyBaseController, MySecondController]


with MyApp() as app:
    app.run()
