package main

import (
	"log"

	zmq "github.com/pebbe/zmq4"
)

func runLoop(zmqSocket *zmq.Socket) {
	for index := range 10 {
		sendMessage := "Hello"

		log.Printf("send(%d): %s\n", index, sendMessage)

		// Send is non-blocking: it enqueues the message in ZMQ's internal
		// send buffer and returns immediately. If the server is absent the
		// message is held in the buffer until the connection is established.
		// If the buffer is full (ZMQ_SNDHWM, default 1000 messages), Send
		// blocks until space is available (flag 0). With DONTWAIT it would
		// return EAGAIN instead. REQ enforces strict send→recv alternation:
		// calling Send twice in a row returns EFSM.
		bytes, err := zmqSocket.Send(sendMessage, 0)
		if err != nil {
			log.Printf("error: send: %v\n", err)

			break
		}

		log.Printf("sent bytes: %d\n", bytes)

		log.Println("receive")

		// Recv blocks until the server sends a reply. If the server is absent,
		// crashed, or never processes the request, this call blocks forever —
		// there is no built-in timeout unless ZMQ_RCVTIMEO is set.
		// If the receive buffer is full (ZMQ_RCVHWM, default 1000 messages),
		// ZMQ drops incoming messages silently — in REQ/REP this is unlikely
		// since at most one reply is in flight at a time.
		receiveMessage, err := zmqSocket.Recv(0)
		if err != nil {
			log.Printf("error: receive: %v\n", err)

			break
		}

		log.Printf("received: %s\n", receiveMessage)
	}
}

func main() {
	log.Println("first-example-client: start.")
	defer log.Println("first-example-client: stop.")

	zmqMajorVer, zmqMinorVer, zmqPatchVer := zmq.Version()

	log.Printf("ZMQ version: %d.%d.%d\n", zmqMajorVer, zmqMinorVer, zmqPatchVer)

	log.Println("create zmq context")

	zmqCtx, err := zmq.NewContext()
	if err != nil {
		log.Printf("error: create zmq context: %v\n", err)

		return
	}

	defer func() {
		err = zmqCtx.Term()
		if err != nil {
			log.Printf("error: terminate zmq context: %v\n", err)
		}
	}()

	log.Println("create zmq socket")

	zmqSocket, err := zmqCtx.NewSocket(zmq.REQ)
	if err != nil {
		log.Printf("error: create zmq socket: %v\n", err)

		return
	}

	defer func() {
		err = zmqSocket.Close()
		if err != nil {
			log.Printf("error: close zmq socket: %v\n", err)
		}
	}()

	endpoint := "tcp://localhost:5555"

	log.Printf("connect to server: %s\n", endpoint)

	// Connect is non-blocking: it returns immediately regardless of whether
	// the server is running. ZMQ establishes the TCP connection in the
	// background (lazy connect). If the server is absent, ZMQ keeps retrying
	// silently — outgoing messages are buffered until the connection is ready.
	err = zmqSocket.Connect(endpoint)
	if err != nil {
		log.Printf("error: connect to server: %v\n", err)

		return
	}

	log.Println("start client loop")

	runLoop(zmqSocket)

	log.Println("stop client loop")
}
