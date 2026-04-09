package main

import (
	"log"
	"time"

	zmq "github.com/pebbe/zmq4"
)

func main() {
	log.Println("first-example-server: start.")
	defer log.Println("first-example-server: stop.")

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

	zmqSocket, err := zmqCtx.NewSocket(zmq.REP)
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

	endpoint := "tcp://*:5555"

	log.Printf("bind zmq socket: %s\n", endpoint)

	err = zmqSocket.Bind(endpoint)
	if err != nil {
		log.Printf("error: bind socket: %v\n", err)

		return
	}

	log.Println("start server loop")

	for {
		log.Println("receive")

		// Recv blocks until a client sends a message.
		// If no client is connected or all clients have crashed before sending,
		// this call blocks forever — there is no timeout and no way to unblock it.
		msg, err := zmqSocket.Recv(0)
		if err != nil {
			log.Printf("error: receive: %v\n", err)

			break
		}

		log.Printf("received: %s\n", msg)

		log.Println("simulate work")

		time.Sleep(time.Second * 10)

		sendMessage := "World"

		log.Printf("send: %s\n", sendMessage)

		// Send returns immediately and queues the reply in ZMQ's internal buffer.
		// If the client disconnected or crashed during the work simulation above,
		// this call still succeeds — no error is returned and the reply is silently lost.
		// The REQ/REP state machine then advances to waiting for the next request,
		// but the client (if it reconnects) will be out of sync with the server state.
		n, err := zmqSocket.Send(sendMessage, 0)
		if err != nil {
			log.Printf("error: send: %v\n", err)

			break
		}

		log.Printf("sent bytes: %d\n", n)
	}

	log.Println("stop server loop")
}
