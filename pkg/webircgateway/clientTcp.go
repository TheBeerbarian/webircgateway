package webircgateway

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

func tcpStartHandler(lAddr string) {
	l, err := net.Listen("tcp", lAddr)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("TCP listening on " + lAddr)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go tcpHandleConn(conn)
	}
}

func tcpHandleConn(conn net.Conn) {
	client := NewClient()

	client.RemoteAddr = conn.RemoteAddr().String()

	clientHostnames, err := net.LookupAddr(client.RemoteAddr)
	if err != nil {
		client.RemoteHostname = client.RemoteAddr
	} else {
		// FQDNs include a . at the end. Strip it out
		potentialHostname := strings.Trim(clientHostnames[0], ".")

		// Must check that the resolved hostname also resolves back to the users IP
		addr, err := net.LookupIP(potentialHostname)
		if err == nil && len(addr) == 1 && addr[0].String() == client.RemoteAddr {
			client.RemoteHostname = potentialHostname
		} else {
			client.RemoteHostname = client.RemoteAddr
		}
	}

	_, remoteAddrPort, _ := net.SplitHostPort(conn.RemoteAddr().String())
	client.Tags["remote-port"] = remoteAddrPort

	client.Log(2, "New client from %s %s", client.RemoteAddr, client.RemoteHostname)

	// We wait until the client send queue has been drained
	var sendDrained sync.WaitGroup
	sendDrained.Add(1)

	// Read from TCP
	go func() {
		for {
			r := make([]byte, 1024)
			len, err := conn.Read(r)
			if err == nil && len > 0 {
				message := string(r[:len])
				message = strings.TrimRight(message, "\r\n")
				client.Log(1, "client->: %s", message)
				select {
				case client.Recv <- message:
				default:
					client.Log(3, "Recv queue full. Dropping data")
					// TODO: Should this really just drop the data or close the connection?
				}

			} else if err != nil {
				client.Log(1, "TCP connection closed (%s)", err.Error())
				break

			} else if len == 0 {
				client.Log(1, "Got 0 bytes from TCP")
			}
		}

		close(client.Recv)
		client.StartShutdown("client_closed")
	}()

	// Process signals for the client
	for {
		signal, ok := <-client.Signals
		if !ok {
			sendDrained.Done()
			break
		}

		if signal[0] == "data" {
			//line := strings.Trim(signal[1], "\r\n")
			line := signal[1] + "\n"
			client.Log(1, "->tcp: %s", line)
			conn.Write([]byte(line))
		}
	}

	sendDrained.Wait()
	conn.Close()
}
