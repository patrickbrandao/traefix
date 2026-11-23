"""A UDP client for the mesh tests.

usage: udp_client.py [<host>:]<port> [size ...]     sends one packet of each size, prints the server and whether it came back whole
       udp_client.py <port> --count <n>    sends n packets of 100 bytes, one every 20 ms, prints the servers that answered
"""
import collections, os, socket, sys

host, _, port = sys.argv[1].rpartition(":")
host, port = host or "127.0.0.1", int(port)
conn = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
conn.connect((host, port))
conn.settimeout(2)
local = "%s:%d" % conn.getsockname()

def ask(payload):
    conn.send(payload)
    try:
        answer = conn.recv(65535)
    except socket.timeout:
        return None, None
    head, _, body = answer.partition(b"|")
    return head.decode().strip(), body

if "--count" in sys.argv:
    import time
    n = int(sys.argv[sys.argv.index("--count") + 1])
    servers = collections.Counter()
    for i in range(n):
        head, _ = ask(os.urandom(100))
        servers[head.split()[0] if head else "sem resposta"] += 1
        time.sleep(0.02)
    print(", ".join("%s: %d" % kv for kv in servers.items()))
else:
    results, server = [], "-"
    for size in [int(x) for x in sys.argv[2:]] or [100]:
        payload = os.urandom(size)
        head, body = ask(payload)
        if head is None:
            results.append("%dB sem resposta" % size)
        else:
            results.append("%dB %s" % (size, "inteiro" if body == payload else "DIFERENTE"))
            server = head
    print("cliente %s | %s | %s" % (local, server, ", ".join(results)))
