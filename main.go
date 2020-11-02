package main

import (
	"flag"
	"fmt"
	"github.com/enitt-dev/go-utils/convert"
	"github.com/stianeikeland/go-rpio"
	"github.com/tarm/serial"
	"log"
	"net"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"
	"encoding/json"
)
const (
	gwState = rpio.Pin(17)
	lan9514 = rpio.Pin(20)
	lan9512 = rpio.Pin(21)
	CTL_1 = rpio.Pin(36)
	CTL_2 = rpio.Pin(37)
	rs485A = rpio.Pin(22)
	wireless = rpio.Pin(25)
)
func LedTest(wg *sync.WaitGroup){
	if err := rpio.Open(); err!= nil{
		fmt.Println(err)
		os.Exit(1)
	}
	gwState.Output()
	lan9514.Output()
	lan9512.Output()
	rs485A.Output()
	CTL_1.Output()
	CTL_2.Output()
	wireless.Output()

	for{    //GPIO LED Toggle
		gwState.Toggle()
		lan9514.Toggle()
		lan9512.Toggle()
		rs485A.Toggle()
		CTL_1.Toggle()
		CTL_2.Toggle()
		wireless.Toggle()
		time.Sleep(time.Second)
	}
}
var sensorTypeMap = map[int]string{
	0x10: "CO2",
	0x11: "TVOC",
	0x20: "Humidity",
	0x30: "Temperature",
	0x31: "Temperature_object",
	0x40: "GyroX",
	0x41: "GyroY",
	0x42: "GyroZ",
}

type sensorData struct {
	DeviceId   uint    `json:"deviceId"`
	SensorType string  `json:"sensor"`
	Value      float32 `json:"value"`
}

func SerialTest(wg *sync.WaitGroup) {
	rs485_A := &serial.Config{Name: "/dev/ttyUSB0", Baud: 115200, StopBits: 1, Parity: 'N'}
	ble := &serial.Config{Name: "/dev/ttyUSB2", Baud: 115200, StopBits: 1, Parity: 'N'}

	a, err := serial.OpenPort(rs485_A)
	if err != nil {
		log.Fatal(err)
	}
	stream, err := serial.OpenPort(ble)
	if err != nil {
		log.Fatal(err)
	}
	var (
		validatorQ = make(chan []byte)
	)

	go recv(validatorQ, stream)

	fmt.Println("Serial port Open")
	for {
		select {
		case recvBytes := <-validatorQ:

			log.Println()

			for i := 0; i < len(recvBytes); i = i + 10 {
				bytes := recvBytes[i : i+10]

				sd := sensorData{
					DeviceId:   uint(bytes[1])*10000 + uint(bytes[3]),
					SensorType: sensorTypeMap[int(bytes[4])],
					Value:      convert.BytesToFloat32(bytes[6:]),
				}

				b, _ := json.Marshal(sd)
				j := string(append(b, '\n'))
				log.Print(j)
				writeToFile(j)

				r, err := a.Write([]byte(j))
				if err != nil {
					log.Fatal(r)
				}
			}
		}
	}
}
func writeToFile(msg string) {
	const fileName = "/home/pi/sensorlog/sensorData6.log"

	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(msg); err != nil {
		panic(err)
	}
}

func EthernetTest01(wg *sync.WaitGroup) {
	port := flag.Int("port01", 3333, "Port to accept connections on.")
	flag.Parse()

	l, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if err != nil {
		log.Panicln(err)
	}
	log.Println("Listening to connections at on port", strconv.Itoa(*port))
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Panicln(err)
		}

		handleRequest(conn)
	}
}
func EthernetTest02(wg *sync.WaitGroup) {
	port := flag.Int("port02", 3334, "Port to accept connections on.")
	flag.Parse()

	l, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if err != nil {
		log.Panicln(err)
	}
	log.Println("Listening to connections at on port", strconv.Itoa(*port))
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Panicln(err)
		}

		handleRequest(conn)
	}
}
func handleRequest(conn net.Conn) {
	log.Println("Accepted new connection.")

	for {
		buf := make([]byte, 1024)
		size, err := conn.Read(buf)
		if err != nil {
			return
		}

		data := buf[:size]
		fmt.Println(reflect.TypeOf(data))
		log.Println("Read new data from connection", data)
		conn.Write(append([]byte(conn.LocalAddr().String()), append([]byte("    "), data...)...))

	}
}
func main() {
	//gpio pin setting
	//Ethernet TCP setting
	//Board Test
	//Led start
	//Rs485 start
	//Ethernet
	var wg sync.WaitGroup

	log.Println("start led toggle")
	wg.Add(4)
	go LedTest(&wg)

	log.Println("start Serial server")
	//wg.Add(2)
	go SerialTest(&wg)

	log.Println("start tcp server")
	//wg.Add(3)
	go EthernetTest01(&wg)
	go EthernetTest02(&wg)

	wg.Wait()
}


