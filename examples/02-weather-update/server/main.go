// Weather update server
// Binds PUB socket to tcp://*:5556
// Publishes random weather updates
package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"

	zmq "github.com/pebbe/zmq4"
)

const (
	maxZipcode       int64 = 100000
	temperatureRange int64 = 215
	temperatureMin   int64 = -80
	humidityRange    int64 = 50
	humidityMin      int64 = 10
)

func randInt64(bound int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(bound))
	if err != nil {
		return 0, fmt.Errorf("generate random int: %w", err)
	}

	return n.Int64(), nil
}

func bindEndpoints(zmqSocket *zmq.Socket, endpoints ...string) error {
	for _, endpoint := range endpoints {
		log.Printf("bind zmq socket: %s\n", endpoint)

		err := zmqSocket.Bind(endpoint)
		if err != nil {
			return fmt.Errorf("bind socket %s: %w", endpoint, err)
		}
	}

	return nil
}

func runLoop(zmqSocket *zmq.Socket) {
	for {
		zipcode, err := randInt64(maxZipcode)
		if err != nil {
			log.Printf("error: generate zipcode: %v\n", err)

			break
		}

		temperature, err := randInt64(temperatureRange)
		if err != nil {
			log.Printf("error: generate temperature: %v\n", err)

			break
		}

		temperature += temperatureMin

		humidity, err := randInt64(humidityRange)
		if err != nil {
			log.Printf("error: generate humidity: %v\n", err)

			break
		}

		humidity += humidityMin

		sendMessage := fmt.Sprintf("%d %d %d", zipcode, temperature, humidity)

		log.Printf("send: %s\n", sendMessage)

		bytes, err := zmqSocket.Send(sendMessage, 0)
		if err != nil {
			log.Printf("error: send: %v\n", err)

			break
		}

		log.Printf("sent bytes: %d\n", bytes)
	}
}

func main() {
	log.Println("weather-update-server: start.")
	defer log.Println("weather-update-server: stop.")

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

	zmqSocket, err := zmqCtx.NewSocket(zmq.PUB)
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

	err = bindEndpoints(zmqSocket, "tcp://*:5555", "ipc://weather.ipc")
	if err != nil {
		log.Printf("error: %v\n", err)

		return
	}

	log.Println("start server loop")

	runLoop(zmqSocket)

	log.Println("stop server loop")
}
