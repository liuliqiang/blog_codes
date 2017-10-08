#!/usr/bin/env python
# encoding: utf-8
import logging
import logging.handlers


class SmartBufferHandler(logging.handlers.MemoryHandler):
    def __init__(self, num_buffered, *args, **kwargs):
        kwargs["capacity"] = num_buffered + 2  # +2 one for current, one for prepop
        super().__init__(*args, **kwargs)

    def emit(self, record):
        if len(self.buffer) == self.capacity - 1:
            self.buffer.pop(0)
        super().emit(record)

handler = SmartBufferHandler(num_buffered=2, target=logging.StreamHandler(), flushLevel=logging.ERROR)
logger = logging.getLogger(__name__)
logger.setLevel("DEBUG")
logger.addHandler(handler)

logger.error("Hello1")
logger.debug("Hello2")  # This line won't be logged
logger.debug("Hello3")
logger.debug("Hello4")
logger.error("Hello5")  # As error will flush the buffered logs, the two last debugs will be logged
