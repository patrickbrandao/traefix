"""A NAT for the mesh tests, in UDP: it stands between a satellite and one instance, like a home router.

usage: udp_nat.py <inside port> <target host:port> [<file of the mappings>]

The satellite sends to 127.0.0.1:<inside port> in place of the instance. Each address of the satellite gets an outside
socket of its own (the mapping), which the instance sees it at, and only the packets that come back to a mapping from the
instance pass: the filtering of a symmetric NAT. SIGUSR1 forgets the mappings, as a router that flushes its table.
The current mappings are written to the file, one outside address per line.
"""
import os
import signal
import socket
import sys
import threading

inside_port, target = int(sys.argv[1]), sys.argv[2]
mappings_file = sys.argv[3] if len(sys.argv) > 3 else None
host, port = target.rsplit(":", 1)
target_addr = (host, int(port))

inside = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
inside.bind(("127.0.0.1", inside_port))

lock = threading.Lock()
mappings = {}


def write_mappings():
    if not mappings_file:
        return
    with open(mappings_file + ".tmp", "w") as f:
        for outside in mappings.values():
            f.write("%s:%d\n" % outside.getsockname())
    os.replace(mappings_file + ".tmp", mappings_file)


def read_outside(outside, satellite):
    while True:
        try:
            data, sender = outside.recvfrom(65535)
        except OSError:
            return
        if sender != target_addr:
            continue
        inside.sendto(data, satellite)


def mapping_for(satellite):
    with lock:
        outside = mappings.get(satellite)
        if outside is None:
            outside = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            outside.bind(("127.0.0.1", 0))
            mappings[satellite] = outside
            threading.Thread(target=read_outside, args=(outside, satellite), daemon=True).start()
            write_mappings()
        return outside


def flush(*_):
    with lock:
        for outside in mappings.values():
            outside.close()
        mappings.clear()
        write_mappings()


signal.signal(signal.SIGUSR1, flush)
write_mappings()

while True:
    try:
        data, satellite = inside.recvfrom(65535)
    except InterruptedError:
        continue
    mapping_for(satellite).sendto(data, target_addr)
