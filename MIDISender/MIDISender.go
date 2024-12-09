package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

var MIDIPortName string = "StageSavvy"
var sockFilePath string = "/tmp/stagesavvy.sock"

func main() {
	// Clean sock file if it already exists
	if _, err := os.Stat(sockFilePath); err == nil {
		if err := os.Remove(sockFilePath); err != nil {
			fmt.Println("Error cleaning sock file:", err)
		}
	}

	// Create sock file and listen
	l, err := net.Listen("unix", sockFilePath)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer l.Close()
	fmt.Println("Listening on unix socket file:", sockFilePath)

	// Accept incoming connections
	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection:", err)
		return
	}
	defer conn.Close()

	// Set up MIDI virtual output port
	driver, err := rtmididrv.New()
	if err != nil {
		panic(err)
	}
	defer driver.Close()

	outPort, err := driver.OpenVirtualOut(MIDIPortName)
	if err != nil {
		panic(err)
	}

	message := ""

	// Keep it running forever
	for {
		fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"), "MIDISender: running new iteration")

		// Check if sock file exists
		if _, err := os.Stat(sockFilePath); err == nil {

		} else if os.IsNotExist(err) {
			fmt.Println("File does not exist")
			os.Exit(0)
		} else {
			fmt.Println("Error checking file:", err)
		}

		// Read messages from socket connection
		buffer := make([]byte, 1024)
		_, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Error reading from socket:", err)
			return
		}
		message = string(buffer)
		fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"), "MIDISender: got new message:", message)

		// Stop
		if strings.HasPrefix(message, "Stop") {
			fmt.Println("App stopped.")
			os.Exit(0)
		}

		if message == "" {
			fmt.Println("Starting MIDISender")
		} else {
			fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"), "MIDISender: Previous response was: "+message)
		}

		// Parse message
		messageList := strings.Split(message, " ")
		messageType := messageList[0]

		if messageType == "NoteOn" || messageType == "NoteOff" || messageType == "CC" {
			messageChannel := messageList[1]
			messageKey := messageList[2]
			messageValue := messageList[3]

			messageChannelConv, err := strconv.Atoi(removeNullBytesAndNewLines(messageChannel))
			if err != nil {
				fmt.Println("Error converting string to integer:", err)
				return
			}

			messageKeyConv, err := strconv.Atoi(removeNullBytesAndNewLines(messageKey))
			if err != nil {
				fmt.Println("Error converting string to integer:", err)
				return
			}

			messageValueConv, err := strconv.Atoi(removeNullBytesAndNewLines(messageValue))
			if err != nil {
				fmt.Println("Error converting string to integer:", err)
				return
			}

			messageChannelUint8 := uint8(messageChannelConv)
			messageKeyUint8 := uint8(messageKeyConv)
			messageValueUint8 := uint8(messageValueConv)

			if messageType == "NoteOn" {
				outPort.Send(midi.NoteOn(messageChannelUint8-1, messageKeyUint8, messageValueUint8))
			}

			if messageType == "NoteOff" {
				outPort.Send(midi.NoteOff(messageChannelUint8-1, messageKeyUint8))
			}

			if messageType == "CC" {
				outPort.Send(midi.ControlChange(messageChannelUint8-1, messageKeyUint8, messageValueUint8))
			}

			fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"), "MIDISender: MIDI message has been sent")
		} else {
			fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"), "MIDISender: Incorrect message type provided")
		}
	}
}

func Use(vals ...interface{}) {
	for _, val := range vals {
		_ = val
	}
}

func removeNullBytesAndNewLines(input string) string {
	output := strings.Builder{}

	for _, c := range input {
		if c != '\x00' && c != '\n' {
			output.WriteRune(c)
		}
	}

	return output.String()
}
