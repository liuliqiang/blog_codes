#!/usr/bin/python3
import array
import fcntl
import json
import socket
import struct


def get_eth_ip():
    max_possible = 128  # arbitrary. raise if needed.
    bytes = max_possible * 32
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    names = array.array('B')
    names.frombytes(('\0' * bytes).encode())
    outbytes = struct.unpack('iL', fcntl.ioctl(
        s.fileno(),
        0x8912,  # SIOCGIFCONF
        struct.pack('iL', bytes, names.buffer_info()[0])
    ))[0]
    namestr = names.tobytes()
    rst = dict({})
    for i in range(0, outbytes, 40):
        name = (namestr[i:i + 16]).decode().split('\0', 1)[0]
        ip = format_ip(namestr[i + 20:i + 24])
        rst[name] = ip
    return rst

def format_ip(addr):
    return str(addr[0]) + '.' + \
           str(addr[1]) + '.' + \
           str(addr[2]) + '.' + \
           str(addr[3])

if __name__ == '__main__':
    rst = get_eth_ip()
    print(json.dumps(rst, indent=2))

