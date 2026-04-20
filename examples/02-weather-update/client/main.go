// Weather proxy listens to weather server which is constantly
// emitting weather data
// Connects SUB socket to tcp://localhost:5556
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	zmq "github.com/pebbe/zmq4"
)

const (
	temperaturesCount  int64  = 100
	defaultZipcode     string = "59937"
	messageFieldsCount int    = 3
)

var (
	errMalformedMessage     = errors.New("malformed message")
	errMalformedTemperature = errors.New("malformed temperature")
)

func parseZipcode() (string, error) {
	raw := defaultZipcode

	if len(os.Args) > 1 {
		raw = os.Args[1]
	}

	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid zipcode %q: %w", raw, err)
	}

	return strconv.FormatInt(n, 10), nil
}

func configureSocket(zmqSocket *zmq.Socket, zipcode string) error {
	log.Printf("set subscribe (zipcode): %s\n", zipcode)

	// SetSubscribe is non-blocking: it registers a prefix filter on the socket.
	// Only messages whose content starts with this prefix will be delivered to
	// Recv. Filtering happens inside ZMQ before messages reach the application.
	err := zmqSocket.SetSubscribe(zipcode)
	if err != nil {
		return fmt.Errorf("set subscribe: %w", err)
	}

	log.Println("connect to server")

	// Connect is non-blocking: it returns immediately regardless of whether the
	// server is running. ZMQ establishes the TCP connection in the background
	// (lazy connect). Messages matching the subscription filter are delivered
	// once the connection is established — messages published before the
	// connection is ready are lost, as PUB/SUB has no replay mechanism.
	err = zmqSocket.Connect("tcp://localhost:5556")
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}

	return nil
}

func runLoop(zmqSocket *zmq.Socket) (int64, int64, error) {
	log.Println("start client loop")
	defer log.Println("stop client loop")

	var (
		totalTemperature int64
		samplesReceived  int64
	)

	for range temperaturesCount {
		log.Println("receive")

		// Recv blocks until a message matching the subscription filter arrives.
		// If the server is absent or not yet publishing, this call blocks
		// forever — there is no built-in timeout unless ZMQ_RCVTIMEO is set.
		// If the incoming buffer is full (ZMQ_RCVHWM, default 1000 messages),
		// ZMQ signals the publisher to drop messages for this subscriber —
		// no error is returned, but messages are silently lost.
		receiveMessage, err := zmqSocket.Recv(0)
		if err != nil {
			return 0, 0, fmt.Errorf("receive: %w", err)
		}

		log.Printf("received: %s\n", receiveMessage)

		readMessageDetails := strings.Split(receiveMessage, " ")

		if len(readMessageDetails) < messageFieldsCount {
			return 0, 0, fmt.Errorf("%w: %q", errMalformedMessage, receiveMessage)
		}

		temperature, err := strconv.ParseInt(readMessageDetails[1], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("%w: %q", errMalformedTemperature, readMessageDetails[1])
		}

		totalTemperature += temperature
		samplesReceived++
	}

	return totalTemperature, samplesReceived, nil
}

func logAverage(zipcode string, total, count int64) {
	if count > 0 {
		log.Printf("average temperature for zipcode %s: %d\n", zipcode, total/count)
	} else {
		log.Println("no samples received, cannot compute average temperature")
	}
}

func mainWithError() error {
	zipcode, err := parseZipcode()
	if err != nil {
		return fmt.Errorf("parse zipcode: %w", err)
	}

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

	zmqSocket, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return fmt.Errorf("create zmq socket: %w", err)
	}

	defer func() {
		err = zmqSocket.Close()
		if err != nil {
			log.Printf("error: close zmq socket: %v\n", err)
		}
	}()

	err = configureSocket(zmqSocket, zipcode)
	if err != nil {
		return fmt.Errorf("configure socket: %w", err)
	}

	totalTemperature, samplesReceived, err := runLoop(zmqSocket)
	if err != nil {
		return fmt.Errorf("run loop: %w", err)
	}

	logAverage(zipcode, totalTemperature, samplesReceived)

	return nil
}

func main() {
	log.Println("weather-update-client: start.")
	defer log.Println("weather-update-client: stop.")

	zmqMajorVer, zmqMinorVer, zmqPatchVer := zmq.Version()

	log.Printf("ZMQ version: %d.%d.%d\n", zmqMajorVer, zmqMinorVer, zmqPatchVer)

	err := mainWithError()
	if err != nil {
		log.Printf("error: %v\n", err)
	}
}
