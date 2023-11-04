package bootstrap

import "net"

type NeighbourDatabase map[string][]net.IP

type Bootstrap struct {
	Neighbours NeighbourDatabase
}

func New(filename string) *Bootstrap {
	return &Bootstrap{
		Neighbours: MustParse(filename),
	}
}

/*

Bootstrap:
	1. Provides the respective neighbours for each node.
		1.1. Client pings the bootstrap via TCP.
			1.1.1. Starts the TCP connection.
			1.1.2. Sends a HELLO packet.
			1.1.3. Waits for data to arrive.
		1.2. Bootstrap sends the node's neighbours to the node.
		1.3. Client measures latency, jitter and packet loss.
		1.4. Client send the measured date to the bootstrap.
			1.4.1. Measures and stores the data for each neighbour received.
			1.4.2. Sends a DATA packet.
			1.4.3. Waits for <-finChan.
		1.6. Bootstrap adds it to the graph.
		1.7. Once every node has done this, bootstrap calculates the MST.
		1.8. Bootstrap reports which node each node must report to.
			1.8.1. The MST is done.
			1.8.2. Bootstrap sends to finChan.
			1.8.3. All active TCP connections resume and send the respective neighbour.

Como é que o cliente sabe para onde enviar ?
Vai o bootstrapper.

*/
