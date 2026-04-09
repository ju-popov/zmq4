// Weather proxy listens to weather server which is constantly
// emitting weather data
// Connects SUB socket to tcp://localhost:5556
package main

import (
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

// parseZipcode reads the zipcode from os.Args or uses the default.
// It validates the value by parsing it as an integer and reformatting,
// which breaks the taint chain from user input for log injection purposes.
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

	err := zmqSocket.SetSubscribe(zipcode)
	if err != nil {
		return fmt.Errorf("set subscribe: %w", err)
	}

	log.Println("connect to server")

	err = zmqSocket.Connect("tcp://localhost:5556")
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}

	return nil
}

func runLoop(zmqSocket *zmq.Socket) (int64, int64) {
	var (
		totalTemperature int64
		samplesReceived  int64
	)

	for range temperaturesCount {
		log.Println("receive")

		receiveMessage, err := zmqSocket.Recv(0)
		if err != nil {
			log.Printf("error: receive: %v\n", err)

			break
		}

		log.Printf("received: %s\n", receiveMessage)

		readMessageDetails := strings.Split(receiveMessage, " ")

		if len(readMessageDetails) < messageFieldsCount {
			log.Printf("error: malformed message: %q\n", receiveMessage)

			break
		}

		temperature, err := strconv.ParseInt(readMessageDetails[1], 10, 64)
		if err != nil {
			log.Printf("error: parse temperature: %v\n", err)

			break
		}

		totalTemperature += temperature
		samplesReceived++
	}

	return totalTemperature, samplesReceived
}

func logAverage(zipcode string, total, count int64) {
	if count > 0 {
		log.Printf("average temperature for zipcode %s: %d\n", zipcode, total/count)
	} else {
		log.Println("no samples received, cannot compute average temperature")
	}
}

func main() {
	log.Println("weather-update-client: start.")
	defer log.Println("weather-update-client: stop.")

	zmqMajorVer, zmqMinorVer, zmqPatchVer := zmq.Version()

	log.Printf("ZMQ version: %d.%d.%d\n", zmqMajorVer, zmqMinorVer, zmqPatchVer)

	zipcode, err := parseZipcode()
	if err != nil {
		log.Printf("error: %v\n", err)

		return
	}

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

	zmqSocket, err := zmqCtx.NewSocket(zmq.SUB)
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

	err = configureSocket(zmqSocket, zipcode)
	if err != nil {
		log.Printf("error: %v\n", err)

		return
	}

	log.Println("start client loop")

	totalTemperature, samplesReceived := runLoop(zmqSocket)

	log.Println("stop client loop")

	logAverage(zipcode, totalTemperature, samplesReceived)
}
