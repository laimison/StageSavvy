package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)

var sockFilePath string = "/tmp/stagesavvy.sock"

func main() {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"), "MIDIMessenger: Running MIDI Messenger")

	args := os.Args[1:]

	if len(args) != 4 {
		fmt.Println("\nUsage:\n./" + filepath.Base(os.Args[0]) + " CC 1 1 1")
		return
	}

	arg1 := args[0]
	arg2 := args[1]
	arg3 := args[2]
	arg4 := args[3]

	message := []byte(arg1 + " " + arg2 + " " + arg3 + " " + arg4)

	conn, err := net.Dial("unix", sockFilePath)
	if err != nil {
		log.Fatalf("Error connecting to socket file: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write(message)
	if err != nil {
		log.Fatalf("Error sending message: %v", err)
	}

	fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"), "MIDIMessenger: Message sent successfully")
}

func Use(vals ...interface{}) {
	for _, val := range vals {
		_ = val
	}
}
