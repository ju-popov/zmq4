package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	zmq "github.com/pebbe/zmq4"
)

const (
	taskCount       = 100
	maxWorkloadMsec = 100
)

func randInt64(bound int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(bound))
	if err != nil {
		return 0, fmt.Errorf("generate random int: %w", err)
	}

	return n.Int64(), nil
}

func runBatch(zmqSenderSocket, zmqSyncSocket *zmq.Socket) error {
	log.Println("start run batch")
	defer log.Println("stop run batch")

	log.Print("Press Enter when the workers are ready: ")

	_, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	syncSendMessage := "0"

	log.Printf("sync send: %s\n", syncSendMessage)

	// Send is non-blocking: it enqueues the start signal in ZMQ's internal
	// send buffer and returns immediately. If the sink is not yet connected,
	// the message is held in the buffer until the connection is ready. If
	// the buffer is full (ZMQ_SNDHWM, default 1000 messages), Send blocks
	// until space is available (flag 0).
	bytes, err := zmqSyncSocket.Send(syncSendMessage, 0)
	if err != nil {
		return fmt.Errorf("sync send: %w", err)
	}

	log.Printf("sync sent bytes: %d\n", bytes)

	var totalMsec int64

	for range taskCount {
		workload, err := randInt64(maxWorkloadMsec)
		if err != nil {
			return fmt.Errorf("randInt64: %w", err)
		}

		totalMsec += workload

		senderSendMessage := strconv.FormatInt(workload, 10)

		log.Printf("sender send: %s\n", senderSendMessage)

		// Send is non-blocking: it enqueues the task in ZMQ's internal send
		// buffer and returns immediately. Tasks are distributed round-robin
		// across connected worker PULL sockets. If no worker is connected,
		// the task is held in the buffer until one connects. If the buffer
		// is full (ZMQ_SNDHWM, default 1000 messages), Send blocks until
		// space is available (flag 0).
		bytes, err = zmqSenderSocket.Send(senderSendMessage, 0)
		if err != nil {
			return fmt.Errorf("sender send: %w", err)
		}

		log.Printf("sender sent bytes: %d\n", bytes)
	}

	log.Printf("Total expected cost: %d msec\n", totalMsec)

	time.Sleep(time.Second)

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

	// Socket to send messages On
	zmqSenderSocket, err := zmqCtx.NewSocket(zmq.PUSH)
	if err != nil {
		return fmt.Errorf("create zmq sender socket: %w", err)
	}

	// Bind is non-blocking: it registers the endpoint immediately and returns.
	// Workers may connect at any time after this call.
	err = zmqSenderSocket.Bind("tcp://*:5557")
	if err != nil {
		return fmt.Errorf("bind zmq sender socket: %w", err)
	}

	defer func() {
		err = zmqSenderSocket.Close()
		if err != nil {
			log.Printf("error: close zmq sender socket: %v\n", err)
		}
	}()

	//  Socket to send start of batch message on
	zmqSyncSocket, err := zmqCtx.NewSocket(zmq.PUSH)
	if err != nil {
		return fmt.Errorf("create zmq sync socket: %w", err)
	}

	// Connect is non-blocking: it returns immediately regardless of whether
	// the sink is running. ZMQ establishes the TCP connection in the
	// background (lazy connect). If the sink is absent, outgoing messages
	// are held in the buffer until the connection is ready.
	err = zmqSyncSocket.Connect("tcp://localhost:5558")
	if err != nil {
		return fmt.Errorf("connect zmq sync socket: %w", err)
	}

	defer func() {
		err = zmqSyncSocket.Close()
		if err != nil {
			log.Printf("error: close zmq sync socket: %v\n", err)
		}
	}()

	err = runBatch(zmqSenderSocket, zmqSyncSocket)
	if err != nil {
		return fmt.Errorf("run batch: %w", err)
	}

	return nil
}

func main() {
	log.Println("parallel-task-ventilator: start.")
	defer log.Println("parallel-task-ventilator: stop.")

	zmqMajorVer, zmqMinorVer, zmqPatchVer := zmq.Version()

	log.Printf("ZMQ version: %d.%d.%d\n", zmqMajorVer, zmqMinorVer, zmqPatchVer)

	err := mainWithError()
	if err != nil {
		log.Printf("error: %v\n", err)
	}
}
