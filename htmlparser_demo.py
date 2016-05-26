#!/usr/bin/env python
# encoding: utf-8
from HTMLParser import HTMLParser


class Status:
    INIT = 0x01
    SUMMARY = 0x02
    REFERRING_SITES = 0x04
    COUNTRIES = 0x08
    TYPES = 0x10
    PLATFORMS = 0x20
    TERMINAL = 0x40

# create a subclass and override the handler methods
class MyHTMLParser(HTMLParser):
    def __init__(self):
        HTMLParser.__init__(self)
        self.status = Status.INIT
        self.tmp = []
        self.count = 0
        self.status_processer = {
            Status.INIT: lambda x: x,
            Status.SUMMARY: self.process_summary_status,
            Status.REFERRING_SITES: self.process_referring_sites_status,
            Status.COUNTRIES: self.process_countries_status,
            Status.TYPES: self.process_types_status,
            Status.PLATFORMS: self.process_platforms_status,
            Status.TERMINAL: self.process_terminal_status
        }

    def transfer_status(self, content):
        if self.status == Status.INIT and content == "Page Views":
            self.count = 0
            self.status = Status.SUMMARY
        elif self.status == Status.SUMMARY and content == "Referring sites":
            self.count = 0
            self.status = Status.REFERRING_SITES
        elif self.status == Status.REFERRING_SITES and content == "Countries":
            self.count = 0
            self.status = Status.COUNTRIES
        elif self.status == Status.COUNTRIES and content == "Types":
            self.count = 0
            self.status = Status.TYPES
        elif self.status == Status.TYPES and content == "Platforms":
            self.count = 0
            self.status = Status.PLATFORMS

    def process_normal(self, missing_upper, content):
        if self.count == 0:
            self.tmp = []
        self.count += 1
        if self.count < missing_upper:
            return True
        self.tmp.append(content)
        return False

    def print_status(self, format):
        print format.format(*self.tmp)
        self.tmp = []

    def process_summary_status(self, content):
        if self.process_normal(4, content):
            return

        if self.count == 6:
            self.print_status("PV: {0}, UV: {1} DL: {2}")

    def process_referring_sites_status(self, content):
        if self.process_normal(5, content):
            return

        if self.count == 7:
            self.print_status("Site: {0}, View: {1}, UV: {2}")

    def process_countries_status(self, content):
        if self.process_normal(5, content):
            return

        if (self.count - 5) % 3 == 2:
            self.print_status("CT: {0}, PV: {1}, UV: {2}")

    def process_types_status(self, content):
        if self.process_normal(5, content):
            return

        if (self.count - 5) % 3 == 2:
            self.print_status("TY: {0}, PV: {1}, UV: {2}")

    def process_platforms_status(self, content):
        if self.process_normal(5, content):
            return

        if (self.count - 5) % 3 == 2:
            self.print_status("PF: {0}, PV: {1}, UV: {2}")

    def process_terminal_status(self, content):
        self.count += 1

    def handle_data(self, data):
        content = data.strip()
        if content:
            self.transfer_status(content)
            self.status_processer[self.status](content)


# instantiate the parser and fed it some HTML
parser = MyHTMLParser()
parser.feed(data)
