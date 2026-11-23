"""A TCP client for the mesh tests.

usage: tcp_client.py [<host>:]<port> [--tls <server name> [--ca <file>]] [--hold <seconds>]
  prints: <local address> | <greeting of the server> [| <issuer of the certificate>]
  --tls   speaks TLS with the server name; with --ca the certificate must be signed by it, otherwise it is not verified
  --hold  keeps the connection, sends a line every 200 ms, and prints how many came back and how the connection ended
"""
import socket, ssl, sys, time

host, _, port = sys.argv[1].rpartition(":")
host, port = host or "127.0.0.1", int(port)

def readline(conn):
    line = b""
    while not line.endswith(b"\n"):
        chunk = conn.recv(1)
        if not chunk:
            return None
        line += chunk
    return line.decode().strip()

try:
    conn = socket.create_connection((host, port), timeout=5)
    local = "%s:%d" % conn.getsockname()
    issuer = None
    if "--tls" in sys.argv:
        server_name = sys.argv[sys.argv.index("--tls") + 1]
        if "--ca" in sys.argv:
            context = ssl.create_default_context(cafile=sys.argv[sys.argv.index("--ca") + 1])
            issuer = "verificado pela CA do servidor"
        else:
            context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
            context.check_hostname = False
            context.verify_mode = ssl.CERT_NONE
        conn = context.wrap_socket(conn, server_hostname=server_name)
    greeting = readline(conn)
    if greeting is None:
        print("%s | conexao fechada sem resposta" % local)
        sys.exit(0)
    print(" | ".join(x for x in (local, greeting, issuer) if x))
    if "--hold" in sys.argv:
        seconds = float(sys.argv[sys.argv.index("--hold") + 1])
        echoed, end, start = 0, "aberta ate o fim", time.time()
        while time.time() - start < seconds:
            try:
                conn.sendall(b"%d\n" % echoed)
                if readline(conn) != "echo %d" % echoed:
                    end = "fechada apos %.1fs" % (time.time() - start)
                    break
                echoed += 1
            except Exception as e:
                end = "erro apos %.1fs (%s)" % (time.time() - start, type(e).__name__)
                break
            time.sleep(0.2)
        print("  %d linhas ecoadas, conexao %s" % (echoed, end))
    conn.close()
except Exception as e:
    print("falhou: %s: %s" % (type(e).__name__, e))
