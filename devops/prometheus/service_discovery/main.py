#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import os
import time
import yaml
import threading

from flask import Flask, request, jsonify
import requests

from meta_manager import ServiceManager
from prom_config_generator import PromConfigGenerator


mongodb_host = os.environ.get("MONGODB_HOST", "localhost")
mongodb_port = os.environ.get("MONGODB_PORT", 27017)
prom_cfg_dir = os.environ.get("PROMETHEUS_CONFIG_DIR", "/etc/prometheus/")
prom_addr = os.environ.get("PROMETHEUS_ADDR", "http://localhost:9090")

app = Flask(__name__)
smm = ServiceManager({
    "host": mongodb_host,
    "port": mongodb_port,
})
cfg_gen = PromConfigGenerator(
    smm, 
    cfg_dir=prom_cfg_dir,
    prom_addr=prom_addr,
)

@app.get("/services")
def list_services():
    """
    list all the services
    """
    services = smm.list_services()
    print(services)
    return jsonify(services)

@app.get("/services/<service_name>")
def get_service(service_name):
    """
    get service by name
    """
    return jsonify(smm.get_service(service_name))

@app.post("/services")
def create_service():
    """
    create service
    """
    smm.create_services(request.json)
    cfg_gen.regen_configs()
    return jsonify({"status": "ok"})

@app.put("/services/<service_name>")
def update_service(service_name):
    """
    update service
    """
    serv = request.json
    if serv is None:
        raise Exception("service is required")
    serv["name"] = service_name
    smm.update_service(serv)
    cfg_gen.regen_configs()
    return jsonify({"status": "ok"})

@app.delete("/services/<service_name>")
def delete_service(service_name):
    """
    delete service
    """
    smm.delete_service(service_name)
    cfg_gen.regen_configs()
    return jsonify({"status": "ok"})

def run_while_loop():
    pass


if __name__ == "__main__":
    # Start the while loop in a separate thread
    # while_loop_thread = threading.Thread(target=run_while_loop)
    # while_loop_thread.start()
    app.run(host="0.0.0.0", port="5555")