package main

import (
	"log"

	zmq "github.com/pebbe/zmq4"
)

func main() {
	log.Println("first-example-client: start.")
	defer log.Println("first-example-client: stop.")

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

			return
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

	// Connect always succeeds immediately regardless of whether the server is running.
	// ZMQ uses lazy connection: the actual TCP handshake happens in the background.
	// If the server is absent, ZMQ will keep retrying silently until it appears.
	err = zmqSocket.Connect(endpoint)
	if err != nil {
		log.Printf("error: connect to server: %v\n", err)

		return
	}

	log.Println("start client loop")

	for index := 0; index < 10; index++ {

		sendMessage := "Hello"

		log.Printf("send(%d): %s\n", index, sendMessage)

		// Send returns immediately and queues the message in ZMQ's internal buffer.
		// If the server is absent or has crashed, the message is held in the buffer
		// and will be delivered once the server comes back — no error is returned.
		n, err := zmqSocket.Send(sendMessage, 0)
		if err != nil {
			log.Printf("error: send: %v\n", err)

			break
		}

		log.Printf("sent bytes: %d\n", n)

		log.Println("receive")

		// Recv blocks until the server sends a reply.
		// If the server is absent, never started, or crashed after receiving the message,
		// this call blocks forever — there is no timeout and no way to unblock it.
		msg, err := zmqSocket.Recv(0)
		if err != nil {
			log.Printf("error: receive: %v\n", err)

			break
		}

		log.Printf("received: %s\n", msg)
	}

	log.Println("stop client loop")
}
