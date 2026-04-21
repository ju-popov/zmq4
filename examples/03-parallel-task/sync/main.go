package main

import (
	"fmt"
	"log"
	"time"

	zmq "github.com/pebbe/zmq4"
)

const taskCount = 100

func runLoop(zmqReceiverSocket *zmq.Socket) error {
	log.Println("start sync loop")
	defer log.Println("stop sync loop")

	for range taskCount {
		// Recv blocks until a result arrives from any connected worker PUSH
		// socket. If all workers finish or disconnect before sending their
		// result, this call blocks forever — there is no built-in timeout
		// unless ZMQ_RCVTIMEO is set. If the receive buffer is full
		// (ZMQ_RCVHWM, default 1000 messages), ZMQ drops incoming messages
		// silently.
		receiveMessage, err := zmqReceiverSocket.Recv(0)
		if err != nil {
			return fmt.Errorf("receive: %w", err)
		}

		log.Printf("received: %s\n", receiveMessage)
	}

	return nil
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

	// Socket to receive messages on
	zmqReceiverSocket, err := zmqCtx.NewSocket(zmq.PULL)
	if err != nil {
		return fmt.Errorf("create zmq receiver socket: %w", err)
	}

	// Bind is non-blocking: it registers the endpoint immediately and returns.
	// The ventilator and workers may connect at any time after this call.
	err = zmqReceiverSocket.Bind("tcp://*:5558")
	if err != nil {
		return fmt.Errorf("bind zmq receiver socket: %w", err)
	}

	defer func() {
		err = zmqReceiverSocket.Close()
		if err != nil {
			log.Printf("error: close zmq receiver socket: %v\n", err)
		}
	}()

	// Recv blocks until the ventilator sends the start signal. If the
	// ventilator is absent or never sends the signal, this call blocks
	// forever — there is no built-in timeout unless ZMQ_RCVTIMEO is set.
	receiveMessage, err := zmqReceiverSocket.Recv(0)
	if err != nil {
		return fmt.Errorf("receive start: %w", err)
	}

	log.Printf("received: %s\n", receiveMessage)

	startTime := time.Now()

	err = runLoop(zmqReceiverSocket)
	if err != nil {
		return fmt.Errorf("run loop: %w", err)
	}

	log.Printf("elapsed time: %s\n", time.Since(startTime).String())

	return nil
}

func main() {
	log.Println("parallel-task-sync: start.")
	defer log.Println("parallel-task-sync: stop.")

	zmqMajorVer, zmqMinorVer, zmqPatchVer := zmq.Version()

	log.Printf("ZMQ version: %d.%d.%d\n", zmqMajorVer, zmqMinorVer, zmqPatchVer)

	err := mainWithError()
	if err != nil {
		log.Printf("error: %v\n", err)
	}
}
