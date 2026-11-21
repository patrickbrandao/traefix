"""A backend for the mesh tests: it echoes what it received, and speaks a line echo protocol after an Upgrade."""
import json, socket, sys, threading

name, port = sys.argv[1], int(sys.argv[2])

def serve(conn, addr):
    data = b""
    while b"\r\n\r\n" not in data:
        chunk = conn.recv(65536)
        if not chunk:
            conn.close(); return
        data += chunk
    head, rest = data.split(b"\r\n\r\n", 1)
    lines = head.decode("latin1").split("\r\n")
    method, path, _ = lines[0].split(" ", 2)
    headers = {}
    for line in lines[1:]:
        key, _, value = line.partition(":")
        headers[key.strip().lower()] = value.strip()
    if headers.get("upgrade") == "echo":
        conn.sendall(b"HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: echo\r\n\r\n")
        buf = rest
        while True:
            while b"\n" in buf:
                line, buf = buf.split(b"\n", 1)
                conn.sendall(b"echo " + name.encode() + b" " + line + b"\n")
            chunk = conn.recv(65536)
            if not chunk:
                conn.close(); return
            buf += chunk
    length = int(headers.get("content-length", "0"))
    body = rest
    while len(body) < length:
        chunk = conn.recv(65536)
        if not chunk: break
        body += chunk
    answer = json.dumps({"backend": name, "peer": addr[0], "method": method, "path": path, "bodyBytes": len(body), "headers": headers}).encode()
    conn.sendall(b"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nConnection: close\r\nContent-Length: " + str(len(answer)).encode() + b"\r\n\r\n" + answer)
    conn.close()

server = socket.socket()
server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
server.bind(("127.0.0.1", port))
server.listen(64)
while True:
    conn, addr = server.accept()
    threading.Thread(target=serve, args=(conn, addr), daemon=True).start()
