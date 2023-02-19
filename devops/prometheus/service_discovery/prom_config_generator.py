#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import os
import time
import yaml
import threading

import requests
from jinja2 import Environment, FileSystemLoader

from meta_manager import ServiceManager


class PromConfigGenerator:
    def __init__(
        self, 
        smm: ServiceManager,
        cfg_dir: str,
        prom_addr: str = "http://localhost:9090",
    ) -> None:
        self.smm = smm
        self.services = []
        self.cfg_dir = cfg_dir
        if not os.path.exists(cfg_dir):
            os.makedirs(cfg_dir)
        self.recording_rules_dir = os.path.join(cfg_dir, "recording_rules")
        if not os.path.exists(self.recording_rules_dir):
            os.makedirs(self.recording_rules_dir)

        self.prom_addr = prom_addr
        if not prom_addr.startswith("http"):
            self.prom_addr = "http://" + prom_addr
    
    def regen_configs(self):
        """
        regenerate the prometheus.yaml config file
        """
        self.services = self.smm.list_services()
        self.gen_prom_config()
        self.gen_recording_rules_file()
        self.reload_prom_config()

    def gen_prom_config(self):
        """
        generator the prometheus.yaml config file
        """
        # Set up Jinja environment
        env = Environment(loader=FileSystemLoader('.'))
        template = env.get_template('templates/prometheus.yaml.tmpl')

        # Render the template with the data
        data = {
            "services": self.services,
        }
        output = template.render(data)

        # write the prometheus config to the file
        main_config_file = os.path.join(self.cfg_dir, "prometheus.yml")
        with open(main_config_file, "w+") as f:
            f.write(output)
    
    def gen_recording_rules_file(self):
        """
        generate the recording rules file
        """
        for service in self.services:
            service_name = service["name"]
            recording_rules_file = os.path.join(self.recording_rules_dir, f"{service_name}.yml")
            with open(recording_rules_file, "w+") as f:
                # todo(liuliqiang): generate the recording rules file
                f.write("")

    def reload_prom_config(self):
        try:
            requests.post("{}/-/reload".format(self.prom_addr))
        except Exception as e:
            print("reload prometheus config failed: {}".format(e))
