"""A TCP server for the mesh tests: it greets with its name and what it knows of the client, then echoes lines.

usage: tcp_backend.py <name> <port> [--proxy] [--tls <cert> <key>]
  --proxy  the connections start with a PROXY protocol v1 line, which tells the address of the client
  --tls    the server terminates TLS itself, for the passthrough routers
"""
import socket, ssl, sys, threading

name, port = sys.argv[1], int(sys.argv[2])
proxy = "--proxy" in sys.argv
context = None
if "--tls" in sys.argv:
    i = sys.argv.index("--tls")
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain(sys.argv[i + 1], sys.argv[i + 2])

def readline(conn):
    line = b""
    while not line.endswith(b"\n"):
        chunk = conn.recv(1)
        if not chunk:
            return None
        line += chunk
    return line

def serve(conn, addr):
    try:
        src = dst = "-"
        if proxy:
            line = readline(conn)
            if line is None:
                return
            parts = line.decode().split()
            src, dst = "%s:%s" % (parts[2], parts[4]), "%s:%s" % (parts[3], parts[5])
        if context:
            conn = context.wrap_socket(conn, server_side=True)
        conn.sendall(("%s src=%s dst=%s\n" % (name, src, dst)).encode())
        while True:
            line = readline(conn)
            if line is None:
                return
            conn.sendall(b"echo " + line)
    except Exception:
        pass
    finally:
        try:
            conn.close()
        except Exception:
            pass

server = socket.socket()
server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
server.bind(("127.0.0.1", port))
server.listen(64)
while True:
    conn, addr = server.accept()
    threading.Thread(target=serve, args=(conn, addr), daemon=True).start()
