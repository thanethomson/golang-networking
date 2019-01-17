package experiments

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"
)

const (
	connectRetries        int           = 3
	clientRetryWait       time.Duration = time.Duration(1) * time.Second
	clientConnectDeadline time.Duration = time.Duration(3) * time.Second
	serverConnectDeadline time.Duration = time.Duration(3) * time.Second
)

func unixSocketAddr() (string, error) {
	f, err := ioutil.TempFile("", "unix-domain-sock-timeouts-*")
	if err != nil {
		return "", err
	}
	addr := f.Name()
	f.Close()
	os.Remove(addr)
	return addr, nil
}

func clientSayHello(conn net.Conn) error {
	if err := conn.SetWriteDeadline(time.Now().Add(clientConnectDeadline)); err != nil {
		return err
	}
	msg := []byte("Hello!")
	n, err := conn.Write(msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("Supposed to write %d bytes, but wrote %d", len(msg), n)
	}
	return nil
}

func clientRecvGreeting(conn net.Conn) ([]byte, error) {
	buf := make([]byte, 50)
	var err error
	for retries := 3; retries > 0; retries-- {
		fmt.Printf("CLIENT\tRead try %d...\n", 4-retries)
		if err = conn.SetReadDeadline(time.Now().Add(clientConnectDeadline)); err != nil {
			return nil, err
		}
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("CLIENT\tFailed to read from server: %s\n", err.Error())
		}
	}
	return buf, err
}

func client(serverAddr string, donec chan struct{}) {
	defer close(donec)
	// try to connect to the server
	for retries := connectRetries; retries > 0; retries-- {
		if retries < connectRetries {
			fmt.Printf("CLIENT\tWaiting to retry...\n")
			// we need this delay, as the Unix domain sockets don't provide any
			// such delay like the TCP sockets do
			time.Sleep(clientRetryWait)
		}

		fmt.Printf("CLIENT\tConnect attempt %d...\n", connectRetries-retries+1)
		conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: serverAddr, Net: "unix"})
		if err != nil {
			fmt.Printf("CLIENT\tClient connect failure with err=%s\n", err.Error())
		} else {
			fmt.Printf("CLIENT\tConnect successful after %d attempts!\n", connectRetries-retries+1)
			if err := clientSayHello(conn); err != nil {
				fmt.Printf("CLIENT\tFailed to say hello to server in time: %s\n", err.Error())
			} else {
				fmt.Printf("CLIENT\tSuccessfully said hello to server\n")
				fmt.Printf("CLIENT\tWaiting for response from server...\n")
				res, err := clientRecvGreeting(conn)
				if err != nil {
					fmt.Printf("CLIENT\tFailed to hear back from server: %s\n", err.Error())
				} else {
					fmt.Printf("CLIENT\tSuccessfully heard back from server: %s\n", string(res))
				}
			}

			if err := conn.Close(); err != nil {
				fmt.Printf("CLIENT\tFailed to close connection: %s\n", err.Error())
			}
			fmt.Printf("CLIENT\tConnection closed.\n")
			break
		}
	}
}

func serverReadGreeting(conn net.Conn) ([]byte, error) {
	if err := conn.SetReadDeadline(time.Now().Add(serverConnectDeadline)); err != nil {
		return nil, err
	}
	buf := make([]byte, 50)
	_, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func serverRespond(conn net.Conn) error {
	if err := conn.SetWriteDeadline(time.Now().Add(serverConnectDeadline)); err != nil {
		return err
	}
	msg := []byte("Hey there!")
	n, err := conn.Write(msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("Supposed to write %d bytes, but wrote %d", len(msg), n)
	}
	fmt.Printf("SERVER\tWrote string: %s", string(msg))
	return nil
}

func serverListenForHelloAndRespond(ln net.Listener, connc chan net.Conn, donec chan struct{}) error {
	defer close(donec)
	conn, err := ln.Accept()
	if err != nil {
		return err
	}
	connc <- conn
	fmt.Printf("SERVER\tClient connected\n")
	// defer func() {
	// 	fmt.Printf("SERVER\tClosing client connection...\n")
	// 	conn.Close()
	// 	fmt.Printf("SERVER\tClosed connection to client.\n")
	// }()

	// try to read some data from the client
	msg, err := serverReadGreeting(conn)
	if err != nil {
		return err
	}
	fmt.Printf("SERVER\tGot greeting from client: %s\n", string(msg))

	// now greet the client back
	fmt.Printf("SERVER\tResponding to client...\n")
	if err := serverRespond(conn); err != nil {
		return err
	}
	fmt.Printf("SERVER\tSuccessfully responded to client.\n")

	return nil
}

func RunUnixDomainSocketTimeoutExperiment() error {
	addr, err := unixSocketAddr()
	if err != nil {
		return err
	}

	fmt.Printf("SERVER\tUsing Unix domain socket: %s\n\n", addr)
	fmt.Printf("------\tTEST 1: Try to connect to dead server\n\n")
	donec := make(chan struct{})
	connc := make(chan net.Conn)
	// first try to connect to a non-existent server
	go client(addr, donec)
	<-donec

	// spin up the server
	ln, err := net.Listen("unix", addr)
	if err != nil {
		return err
	}
	defer func() {
		fmt.Printf("SERVER\tShutting down server...\n")
		ln.Close()
		fmt.Printf("SERVER\tServer shut down.\n")
	}()

	fmt.Printf("\n------\tTEST 2: Connect to listening server, but nobody's accepting\n\n")
	donec = make(chan struct{})
	connc = make(chan net.Conn)
	go client(addr, donec)
	<-donec

	fmt.Printf("\n------\tTEST 3: Connect to listening server, with server accepting\n\n")
	donec = make(chan struct{})
	serverDonec := make(chan struct{})
	connc = make(chan net.Conn)
	go serverListenForHelloAndRespond(ln, connc, serverDonec)
	go client(addr, donec)
	serverConn := <-connc
	defer func() {
		fmt.Printf("SERVER\tClosing client connection...\n")
		serverConn.Close()
		fmt.Printf("SERVER\tClient connection closed.\n")
	}()
	<-serverDonec
	<-donec

	return nil
}
