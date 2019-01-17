# Golang Networking Experiments

This repo, so far, contains one networking experiment with Golang.

## Unix Domain Sockets

### Connect Timeouts
The first experiment, in `cmd/unix_domain_sockets/main.go`, will:

1. Spin up a simple stream server on a Unix domain socket. It will be listening,
   but not accepting connections.
2. Instantiate a client to connect to attempt to connect to that server.
3. Provide reporting information as to what happens, and if timeouts in
   connecting are applicable.