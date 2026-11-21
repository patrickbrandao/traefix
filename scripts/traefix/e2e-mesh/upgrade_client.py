"""Asks for an Upgrade through an entry point, and exchanges two lines."""
import socket, sys
port, host = int(sys.argv[1]), sys.argv[2]
s = socket.create_connection(("127.0.0.1", port)); s.settimeout(10)
s.sendall(("GET /ws HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: echo\r\n\r\n" % host).encode())
data = b""
while b"\r\n\r\n" not in data:
    data += s.recv(65536)
head, rest = data.split(b"\r\n\r\n", 1)
print(head.decode().split("\r\n")[0])
out = rest
for word in (b"one", b"two"):
    s.sendall(word + b"\n")
    while out.count(b"\n") < 1:
        out += s.recv(65536)
    line, out = out.split(b"\n", 1)
    print(line.decode())
