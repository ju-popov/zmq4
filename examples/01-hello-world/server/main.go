package main

import (
	"fmt"
	"log"
	"time"

	zmq "github.com/pebbe/zmq4"
)

func runLoop(zmqSocket *zmq.Socket) error {
	log.Println("start server loop")
	defer log.Println("stop server loop")

	for {
		log.Println("receive")

		// Recv blocks until a client sends a request. If no client is connected
		// or all clients have gone away, this call blocks forever — there is no
		// built-in timeout unless ZMQ_RCVTIMEO is set.
		// If the receive buffer is full (ZMQ_RCVHWM, default 1000 messages),
		// ZMQ drops incoming messages silently — in REQ/REP this is unlikely
		// since REP processes one request at a time.
		// REP enforces strict recv→send alternation: calling Recv twice in a
		// row returns EFSM.
		msg, err := zmqSocket.Recv(0)
		if err != nil {
			return fmt.Errorf("receive: %w", err)
		}

		log.Printf("received: %s\n", msg)

		log.Println("simulate work")

		time.Sleep(time.Second * 1)

		sendMessage := "World"

		log.Printf("send: %s\n", sendMessage)

		// Send is non-blocking: it enqueues the reply in ZMQ's internal send
		// buffer and returns immediately. If the client disconnected during the
		// work simulation above, Send still succeeds — ZMQ detects the gone
		// peer and silently drops the message. No error is returned.
		// If the buffer is full (ZMQ_SNDHWM, default 1000 messages), Send
		// blocks until space is available (flag 0). In REQ/REP this is unlikely
		// since at most one reply per client is in flight at a time.
		bytes, err := zmqSocket.Send(sendMessage, 0)
		if err != nil {
			return fmt.Errorf("send: %w", err)
		}

		log.Printf("sent bytes: %d\n", bytes)
	}
}

func mainWithError() error {
	log.Println("create zmq context")

	zmqCtx, err := zmq.NewContext()
	if err != nil {
		return fmt.Errorf("create zmq context: %w", err)
	}

	defer func() {
		err = zmqCtx.Term()
		if err != nil {
			log.Printf("error: terminate zmq context: %v\n", err)
		}
	}()

	log.Println("create zmq socket")

	zmqSocket, err := zmqCtx.NewSocket(zmq.REP)
	if err != nil {
		return fmt.Errorf("create zmq socket: %w", err)
	}

	defer func() {
		err = zmqSocket.Close()
		if err != nil {
			log.Printf("error: close zmq socket: %v\n", err)
		}
	}()

	endpoint := "tcp://*:5555"

	log.Printf("bind zmq socket: %s\n", endpoint)

	// Bind is non-blocking: it registers the endpoint immediately and returns.
	// Clients may connect at any time after this call.
	err = zmqSocket.Bind(endpoint)
	if err != nil {
		return fmt.Errorf("bind zmq socket: %w", err)
	}

	err = runLoop(zmqSocket)
	if err != nil {
		return fmt.Errorf("run loop: %w", err)
	}

	return nil
}

func main() {
	log.Println("hello-world-server: start.")
	defer log.Println("hello-world-server: stop.")

	zmqMajorVer, zmqMinorVer, zmqPatchVer := zmq.Version()

	log.Printf("ZMQ version: %d.%d.%d\n", zmqMajorVer, zmqMinorVer, zmqPatchVer)

	err := mainWithError()
	if err != nil {
		log.Printf("error: %v\n", err)
	}
}
