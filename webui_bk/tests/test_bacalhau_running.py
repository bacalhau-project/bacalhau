#!/usr/bin/env python3

# Use python to test to see if server is running on 1234 port
if __name__ == "__main__":
    # If there is a port given as an argument, use that. Otherwise, use 1234
    import sys

    if len(sys.argv) > 1:
        port = sys.argv[1]
    else:
        port = 1234

    import socket
    import sys

    # Create a TCP/IP socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

    # Connect the socket to the port where the server is listening
    server_address = ("localhost", port)

    print("connecting to %s port %s" % server_address)

    # If there is no server running, this will fail
    try:
        sock.connect(server_address)
    except ConnectionRefusedError:
        print("No server running on port %s" % port, file=sys.stderr)
        sys.exit(1)

    try:
        # Send data
        message = "Test message (should be meaningless)."
        print('sending "%s"' % message)
        sock.sendall(bytes(message, "UTF-8"))

        # Look for the response
        amount_received = 0

        # Wait for the response - if nothing received after 3 seconds, fail
        sock.settimeout(3)
        amount_expected = len(message)
        while amount_received < amount_expected:
            try:
                data = sock.recv(16)
                amount_received += len(data)
                print('received "%s"' % data)
            except socket.timeout:
                # Print timeout message to stderr and exit with error code 1
                print("No response received - timeout.", file=sys.stderr)
                sys.exit(1)
            except Exception:
                print("Unexpected error: %s" % sys.exc_info()[0], file=sys.stderr)
                sys.exit(1)

    finally:
        print("closing socket")
        sock.close()
