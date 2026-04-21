package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	zmq "github.com/pebbe/zmq4"
)

var errMalformedMessage = errors.New("malformed message")

func runLoop(zmqReceiverSocket *zmq.Socket, zmqSenderSocket *zmq.Socket) error {
	log.Println("start worker loop")
	defer log.Println("stop worker loop")

	for {
		// Recv blocks until a task arrives from the ventilator's PUSH socket.
		// If the ventilator is absent or has finished sending all tasks, this
		// call blocks forever — there is no built-in timeout unless
		// ZMQ_RCVTIMEO is set. If the receive buffer is full (ZMQ_RCVHWM,
		// default 1000 messages), ZMQ drops incoming messages silently.
		receiveMessage, err := zmqReceiverSocket.Recv(0)
		if err != nil {
			return fmt.Errorf("receive: %w", err)
		}

		log.Printf("received: %s\n", receiveMessage)

		//  Do the work
		msec, err := strconv.ParseInt(receiveMessage, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: %q", errMalformedMessage, receiveMessage)
		}

		time.Sleep(time.Duration(msec) * time.Millisecond)

		sendMessage := ""

		log.Printf("send: %s\n", sendMessage)

		// Send is non-blocking: it enqueues the result in ZMQ's internal send
		// buffer and returns immediately. If the sink disconnects, Send still
		// succeeds — ZMQ detects the gone peer and silently drops the message.
		// If the buffer is full (ZMQ_SNDHWM, default 1000 messages), Send
		// blocks until space is available (flag 0).
		bytes, err := zmqSenderSocket.Send(sendMessage, 0)
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

	// Socket to receive messages on
	zmqReceiverSocket, err := zmqCtx.NewSocket(zmq.PULL)
	if err != nil {
		return fmt.Errorf("create zmq receiver socket: %w", err)
	}

	// Connect is non-blocking: it returns immediately regardless of whether
	// the ventilator is running. ZMQ establishes the TCP connection in the
	// background (lazy connect). If the ventilator is absent, ZMQ keeps
	// retrying silently — incoming tasks are buffered on the ventilator's
	// side until the connection is ready.
	err = zmqReceiverSocket.Connect("tcp://localhost:5557")
	if err != nil {
		return fmt.Errorf("connect zmq receiver socket: %w", err)
	}

	defer func() {
		err = zmqReceiverSocket.Close()
		if err != nil {
			log.Printf("error: close zmq receiver socket: %v\n", err)
		}
	}()

	// Socket to send messages to task sink
	zmqSenderSocket, err := zmqCtx.NewSocket(zmq.PUSH)
	if err != nil {
		return fmt.Errorf("create zmq sender socket: %w", err)
	}

	// Connect is non-blocking: it returns immediately regardless of whether
	// the sink is running. ZMQ establishes the TCP connection in the
	// background (lazy connect). If the sink is absent, outgoing results
	// are held in the buffer until the connection is ready.
	err = zmqSenderSocket.Connect("tcp://localhost:5558")
	if err != nil {
		return fmt.Errorf("connect zmq sender socket: %w", err)
	}

	defer func() {
		err = zmqSenderSocket.Close()
		if err != nil {
			log.Printf("error: close zmq sender socket: %v\n", err)
		}
	}()

	err = runLoop(zmqReceiverSocket, zmqSenderSocket)
	if err != nil {
		return fmt.Errorf("run loop: %w", err)
	}

	return nil
}

func main() {
	log.Println("parallel-task-worker: start.")
	defer log.Println("parallel-task-worker: stop.")

	zmqMajorVer, zmqMinorVer, zmqPatchVer := zmq.Version()

	log.Printf("ZMQ version: %d.%d.%d\n", zmqMajorVer, zmqMinorVer, zmqPatchVer)

	err := mainWithError()
	if err != nil {
		log.Printf("error: %v\n", err)
	}
}
