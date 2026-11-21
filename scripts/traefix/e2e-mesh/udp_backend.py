"""A UDP server for the mesh tests: it answers each packet with its name, the address it came from, and the packet.

usage: udp_backend.py <name> <port>
"""
import socket, sys

name, port = sys.argv[1], int(sys.argv[2])
server = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
server.bind(("127.0.0.1", port))
while True:
    data, addr = server.recvfrom(65535)
    server.sendto(("%s from=%s:%d |" % (name, addr[0], addr[1])).encode() + data, addr)
