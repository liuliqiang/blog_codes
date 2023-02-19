#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import os
import time
import yaml
import threading

from pymongo import MongoClient

class ServiceManager:
    def __init__(self, opts: dict) -> None:
        self.cli = MongoClient(
            opts.get("host", "localhost"), 
            opts.get("port", 27017),
        )

    def list_services(self) -> list:
        """
        list all the services
        """
        find_rst = self.cli.prom_db.service_tab.find({})
        rst = []
        for s in find_rst:
            del(s["_id"])
            rst.append(s)
        return rst

    def get_service(self, service_name: str) -> dict:
        """
        get service by name
        """
        find_rst = self.cli.prom_db.service_tab.find_one({"name": service_name})
        del find_rst["_id"]
        return find_rst
        
    def create_services(self, service: dict):
        if service.get("name") is None:
            raise Exception("service name is required")
        self.cli.prom_db.service_tab.update_one(
            {"service_name": service.get("name"),}, 
            {"$set": service}, 
            upsert=True,
        )
        
    
    def update_service(self, service: dict):
        if service.get("name") is None:
            raise Exception("service name is required")
        self.cli.prom_db.service_tab.update_one(
            {"service_name": service.get("name"),}, 
            {"$set": service}, 
        )

    def delete_service(self, service_name: str):
        self.cli.prom_db.service_tab.delete_one({"service_name": service_name})