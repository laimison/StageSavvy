package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	toml "github.com/pelletier/go-toml/v2"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

var sockFilePath string = "/tmp/stagesavvy.sock"
var tomlFile string
var activeMapping string = "MP1"
var availableMappingList []string
var routingCh chan string = make(chan string)
var wholeTOML map[string]interface{}
var inPortsList []drivers.In
var outPortsList []drivers.Out

func main() {
	// Variables
	use(activeMapping)

	// Check if program is already running
	if isProcessRunning("MIDITranslator") {
		fmt.Println("MIDITranslator program is already running")
		os.Exit(5)
	}

	// Create a channel to receive program interrupt signals
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT)

	// Setup Unix domain socket
	listenToSock, err := setupSocket(sockFilePath)
	must(err)
	defer listenToSock.Close()

	// Accept incoming connections via sock
	go socketReceiverRoutineSender(routingCh, listenToSock)
	go routineReceiver(routingCh)

	// Create MIDI Driver
	driver, err := rtmididrv.New()
	must(err)

	// Close MIDI Driver when program ended
	defer driver.Close()

	// Get script's dir, go one dir back and add Settings.txt
	tomlFile = getTOMLFilePath()

	// Read TOML file
	wholeTOML = readTOMLFile(tomlFile)

	// Get available mappings
	availableMappingList = getMappingNames(wholeTOML)

	// Get MIDI input ports
	inPorts := getMIDIInputPorts(wholeTOML)
	logms("Input Ports:", strings.Join(inPorts, ", "))

	// Open MIDI input ports
	for _, portName := range inPorts {
		for {
			inPort, err := midi.FindInPort(portName)
			if err == nil {
				inPortsList = append(inPortsList, inPort)
				break
			} else {
				logms("Failed to use input port", portName, ". Retrying ...")
			}
			time.Sleep(10 * time.Second)
		}
	}

	// Get MIDI output ports
	outPorts := getMIDIOutputPorts(wholeTOML)
	logms("Output Ports:", strings.Join(outPorts, ", "))

	// Open MIDI output ports
	for _, portName := range outPorts {
		outPort, err := driver.OpenVirtualOut(portName)
		must(err)
		outPortsList = append(outPortsList, outPort)
	}

	// Monitor MIDI input ports for disconnection and reconnection
	go func() {
		inputPortDisappeared := false
		inputPortDisappearedOld := false
		for {
			requiredInPorts := getMIDIInputPorts(wholeTOML)

			// Check if any input port is disconnected
			for _, portName := range requiredInPorts {
				inputPortDisappearedOld = inputPortDisappeared
				_, err := midi.FindInPort(portName)

				if err != nil {
					inputPortDisappeared = true
					logms("Input port", portName, "disappeared. Retrying ...")
				} else {
					inputPortDisappeared = false
				}

				if inputPortDisappearedOld && !inputPortDisappeared {
					logms("Input port", portName, "appeared")
					listenToMIDIMessages(inPortsList, outPortsList, routingCh, wholeTOML)
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()

	// Listen to MIDI messages
	listenToMIDIMessages(inPortsList, outPortsList, routingCh, wholeTOML)

	// infinite loop to listen to MIDI messages forever
	for {
		// Check if sock file exists
		_, err := os.Stat(sockFilePath)
		must(err)

		// Block until a keyboard's signal is received
		<-interruptChan
		logms("\nCTRL+C pressed. Exiting...")
		os.Exit(0)
	}
}
func isProcessRunning(processName string) bool {
	output, err := exec.Command("pgrep", processName).Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}

func listenToMIDIMessages(inPortsList []drivers.In, outPortsList []drivers.Out, routingCh chan<- string, wholeTOML map[string]interface{}) {
	var ch, key, val uint8

	for _, inPort := range inPortsList {
		_, err := midi.ListenTo(inPort, func(msg midi.Message, timestampms int32) {
			switch msg.Type() {
			case midi.ControlChangeMsg:
				msg.GetControlChange(&ch, &key, &val)
				destDevice, destType, destCh, destKey, destVal, destDelay := getBindingValue(routingCh, wholeTOML, activeMapping, inPort.String(), "CC", ch, key, val)

				if destDevice != "" {
					for _, outPort := range outPortsList {
						if destDevice != outPort.String() {
							continue
						}
						translationValue := fmt.Sprintf("%s.%s.%d.%d.%d", destDevice, destType, destCh, destKey, destVal)
						if destType == "CC" {
							logms(fmt.Sprintf("Message received:   "+inPort.String()+".CC.%d.%d.%d", ch, key, val))
							outPort.Send(midi.ControlChange(destCh, destKey, destVal))
							logms("Message translated:", translationValue)
						}

						if destType == "NOTE" {
							logms(fmt.Sprintf("Message received:   "+inPort.String()+".CC.%d.%d.%d", ch, key, val))
							outPort.Send(midi.NoteOn(destCh, destKey, destVal))
							logms("Message translated:", translationValue, "| NoteOn")
							go func() {
								time.Sleep(time.Duration(destDelay) * time.Millisecond)
								outPort.Send(midi.NoteOff(destCh, destKey))
								logms("Message translated:", translationValue, "| NoteOff |", "Delay", destDelay, "ms")
							}()
						}
					}
				}
			case midi.NoteOnMsg:
				msg.GetNoteStart(&ch, &key, &val)
				destDevice, destType, destCh, destKey, destVal, destDelay := getBindingValue(routingCh, wholeTOML, activeMapping, inPort.String(), "NOTEON", ch, key, val)

				if destDevice != "" {
					for _, outPort := range outPortsList {
						if destDevice != outPort.String() {
							continue
						}
						translationValue := fmt.Sprintf("%s.%s.%d.%d.%d", destDevice, destType, destCh, destKey, destVal)
						if destType == "CC" {
							logms(fmt.Sprintf("Message received:   "+inPort.String()+".NOTE(ON).%d.%d.%d", ch, key, val))
							outPort.Send(midi.ControlChange(destCh, destKey, destVal))
							logms("Message translated:", translationValue)
						}

						if destType == "NOTE" {
							logms(fmt.Sprintf("Message received:   "+inPort.String()+".NOTE(ON).%d.%d.%d", ch, key, val))
							outPort.Send(midi.NoteOn(destCh, destKey, destVal))
							logms("Message translated:", translationValue, "| NoteOn")
							go func() {
								time.Sleep(time.Duration(destDelay) * time.Millisecond)
								outPort.Send(midi.NoteOff(destCh, destKey))
								logms("Message translated:", translationValue, "| NoteOff |", "Delay", destDelay, "ms")
							}()
						}
					}
				}
			case midi.NoteOffMsg:
				msg.GetNoteStart(&ch, &key, &val)
				destDevice, destType, destCh, destKey, destVal, destDelay := getBindingValue(routingCh, wholeTOML, activeMapping, inPort.String(), "NOTEOFF", ch, key, val)

				if destDevice != "" {
					for _, outPort := range outPortsList {
						if destDevice != outPort.String() {
							continue
						}
						translationValue := fmt.Sprintf("%s.%s.%d.%d.%d", destDevice, destType, destCh, destKey, destVal)
						if destType == "NOTE" {
							logms(fmt.Sprintf("Message received:   "+inPort.String()+".NOTE(OFF).%d.%d.%d", ch, key, val))
							if destDelay == 0 {
								outPort.Send(midi.NoteOff(destCh, destKey))
								logms("Message translated:", translationValue, "| NoteOff")
							}
						}
					}
				}
			}
		})
		must(err)
	}
}

func socketReceiverRoutineSender(routingCh chan<- string, listenToSock net.Listener) {
	for {
		conn, err := listenToSock.Accept()
		if err != nil {
			logms("Error accepting connection:", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			buf := make([]byte, 1024)
			for {
				n, err := c.Read(buf)
				if err != nil {
					if err != net.ErrClosed {
						logms("Error reading from connection:", err)
					}
					return
				}
				message := string(buf[:n])
				message = strings.TrimSpace(message)

				// MP
				if strings.HasPrefix(message, "MP") {
					routingCh <- message
				}

				// Stop
				if strings.HasPrefix(message, "Stop") {
					logms("App stopped.")
					os.Exit(0)
				}
			}
		}(conn)
	}
}
func routineReceiver(routingCh <-chan string) {
	for msg := range routingCh {
		if strings.HasPrefix(msg, "MP") {
			found := false
			for _, mapping := range availableMappingList {
				if msg == mapping {
					oldMapping := activeMapping
					activeMapping = msg
					logms("Mapping changed from", oldMapping, "to", activeMapping)
					found = true
					break
				}
			}
			// assign activeMapping next available mapping
			activeMappingItemNo := 0
			if msg == "MPN" {
				for i, mapping := range availableMappingList {
					if activeMapping == mapping {
						activeMappingItemNo = i
						break
					}
				}
				oldMapping := activeMapping
				if activeMappingItemNo == len(availableMappingList)-1 {
					activeMapping = availableMappingList[0]
				} else {
					activeMapping = availableMappingList[activeMappingItemNo+1]
				}
				found = true
				logms("Mapping changed from", oldMapping, "to", activeMapping)
			}
			if !found {
				logms("Mapping not found:", msg)
			}
		}
	}
}

func setupSocket(sockFilePath string) (net.Listener, error) {
	// Remove existing socket file if it exists
	if _, err := os.Stat(sockFilePath); err == nil {
		if err := os.Remove(sockFilePath); err != nil {
			return nil, err
		}
	}

	// Create a new Unix domain socket
	addr, err := net.ResolveUnixAddr("unix", sockFilePath)
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func getTOMLFilePath() string {
	pathParts := strings.Split(os.Args[0], "/")
	if len(pathParts) > 2 {
		pathParts = pathParts[:len(pathParts)-2]
	}
	scriptOneDirBack := strings.Join(pathParts, "/")

	cwd, _ := os.Getwd()
	cwdParts := strings.Split(cwd, "/")
	if len(cwdParts) > 1 {
		cwdParts = cwdParts[:len(cwdParts)-1]
	}
	cwdOneDirBack := strings.Join(cwdParts, "/")

	ex, err := os.Executable()
	must(err)
	if strings.HasSuffix(ex, "MIDITranslator/MIDITranslator") {
		tomlFile = scriptOneDirBack + "/Settings.txt"
	} else {
		tomlFile = cwdOneDirBack + "/Settings.txt"
	}

	return tomlFile
}

func readTOMLFile(tomlFile string) map[string]interface{} {
	wholeTOML := make(map[string]interface{})

	content, err := os.ReadFile(tomlFile)
	must(err)

	err = toml.Unmarshal(content, &wholeTOML)
	must(err)

	return wholeTOML
}

func getMIDIInputPorts(wholeTOML map[string]interface{}) []string {
	var MIDIInputPorts []string

	for _, mpContent := range wholeTOML {
		itemKey := make(map[string]interface{})
		itemKeyStr, err := toml.Marshal(mpContent)
		must(err)

		err = toml.Unmarshal(itemKeyStr, &itemKey)
		must(err)

		for itemKey, _ := range itemKey {
			itemKeyFirst := strings.Split(fmt.Sprintf("%v", itemKey), ".")[0]
			MIDIInputPorts = append(MIDIInputPorts, fmt.Sprintf("%v", itemKeyFirst))
		}

		// Remove duplicates
		uniquePorts := make(map[string]bool)
		for _, port := range MIDIInputPorts {
			uniquePorts[port] = true
		}

		MIDIInputPorts = make([]string, 0, len(uniquePorts))
		for port := range uniquePorts {
			MIDIInputPorts = append(MIDIInputPorts, port)
		}
	}

	return MIDIInputPorts
}

func getMIDIOutputPorts(wholeTOML map[string]interface{}) []string {
	var MIDIOutputPorts []string

	for _, mpContent := range wholeTOML {
		itemKey := make(map[string]interface{})
		itemKeyStr, err := toml.Marshal(mpContent)
		must(err)

		err = toml.Unmarshal(itemKeyStr, &itemKey)
		must(err)

		for _, itemValue := range itemKey {
			// Ignore commands
			if strings.HasPrefix(fmt.Sprintf("%v", itemValue), "[") && strings.HasSuffix(fmt.Sprintf("%v", itemValue), "]") {
				continue
			}

			itemValueFirst := strings.Split(fmt.Sprintf("%v", itemValue), ".")[0]
			MIDIOutputPorts = append(MIDIOutputPorts, fmt.Sprintf("%v", itemValueFirst))
		}

		// Remove duplicates
		uniquePorts := make(map[string]bool)
		for _, port := range MIDIOutputPorts {
			uniquePorts[port] = true
		}

		MIDIOutputPorts = make([]string, 0, len(uniquePorts))
		for port := range uniquePorts {
			MIDIOutputPorts = append(MIDIOutputPorts, port)
		}
	}

	return MIDIOutputPorts
}

func getBindingValue(routingCh chan<- string, wholeTOML map[string]interface{}, MIDImapping string, MIDIdevice string, MIDIType string, MIDIchannel uint8, MIDIkey uint8, MIDIvalue uint8) (string, string, uint8, uint8, uint8, uint16) {
	var destDevice string
	var destType string
	var destChannel uint8
	var destKey uint8
	var destValue uint8
	var destDelay uint16

	for mpName, mpContent := range wholeTOML {
		if mpName != MIDImapping {
			continue
		}

		itemKey := make(map[string]interface{})
		itemKeyStr, err := toml.Marshal(mpContent)
		must(err)

		err = toml.Unmarshal(itemKeyStr, &itemKey)
		must(err)

		// Go through each item in a given mapping - check for commands
		if MIDIType == "NOTEON" {
			for item, itemValue := range itemKey {
				itemString := strings.Split(item, ".")
				use(itemString)
				itemValueString := strings.Split(fmt.Sprintf("%v", itemValue), ".")

				if strings.HasPrefix(itemValueString[0], "[") && strings.HasSuffix(itemValueString[0], "]") {

					if itemString[2] == fmt.Sprintf("%d", MIDIchannel) && itemString[3] == fmt.Sprintf("%d", MIDIkey) && itemString[4] == fmt.Sprintf("%d", MIDIvalue) {
						newMapping := strings.Trim(itemValueString[0], "[]")
						routingCh <- newMapping
						return "", "", 0, 0, 0, 0
					}
				}
			}
		}

		// Go through each item in a given mapping - check for MIDI bindings
		for item, itemValue := range itemKey {
			dkSplit := strings.Split(item, ".")

			availableDevice := dkSplit[0]
			availableType := dkSplit[1]
			availableChannel := dkSplit[2]
			availableKey := dkSplit[3]
			availableValue := dkSplit[4]
			use(availableDevice, availableType, availableChannel, availableKey, availableValue)

			itemString := fmt.Sprintf("%v", item)
			itemList := strings.Split(itemString, ".")

			itemValueString := fmt.Sprintf("%v", itemValue)
			itemValueList := strings.Split(itemValueString, ".")

			itemFirstValue := itemValueList[0]

			if strings.HasPrefix(itemFirstValue, "[") && strings.HasSuffix(itemFirstValue, "]") {
				// ignore command in processing
				continue
			}

			destDevice = itemValueList[0]
			destType = itemValueList[1]
			destChannelUint, err := strconv.ParseUint(itemValueList[2], 10, 8)
			must(err)
			destChannel = uint8(destChannelUint)
			destKeyUint, err := strconv.ParseUint(itemValueList[3], 10, 8)
			must(err)
			destKey = uint8(destKeyUint)

			continueOuterLoop := false
			if itemList[4] == "X" {
				for item, itemValue := range itemKey {
					use(itemValue)
					dkSplit := strings.Split(item, ".")
					availableKey := dkSplit[3]
					availableValue := dkSplit[4]
					if availableKey == fmt.Sprintf("%d", MIDIkey) && availableValue == fmt.Sprintf("%d", MIDIvalue) {
						continueOuterLoop = true
						continue
					}
				}
			}

			if continueOuterLoop {
				continue
			}

			if itemValueList[4] == "X" {
				destValue = MIDIvalue
			} else {
				destValueUint, err := strconv.ParseUint(itemValueList[4], 10, 8)
				must(err)
				destValue = uint8(destValueUint)
			}

			if len(itemValueList) == 5 {
				destDelay = 0
			} else {
				delayValue, err := strconv.ParseUint(strings.TrimSuffix(itemValueList[5], "ms"), 10, 16)
				must(err)
				destDelay = uint16(delayValue)
			}

			if availableDevice == MIDIdevice && strings.HasPrefix(MIDIType, availableType) && availableChannel == fmt.Sprintf("%d", MIDIchannel) && availableKey == fmt.Sprintf("%d", MIDIkey) {
				if availableValue == fmt.Sprintf("%d", MIDIvalue) || availableValue == "X" {
					return destDevice, destType, destChannel, destKey, destValue, destDelay
				}
			}
		}
	}

	return "", "", 0, 0, 0, 0
}

func getMappingNames(wholeTOML map[string]interface{}) []string {
	var MIDImappings []string

	for mpName := range wholeTOML {
		MIDImappings = append(MIDImappings, mpName)
	}

	return MIDImappings
}

// Function useful in development
func use(vals ...interface{}) {
	for _, val := range vals {
		_ = val
	}
}

// Close program on failure
func must(err error) {
	if err != nil {
		panic(err.Error())
	}
}

// Log
func logms(logMessages ...interface{}) {
	var strMessages []string
	for _, msg := range logMessages {
		strMessages = append(strMessages, fmt.Sprintf("%v", msg))
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"), strings.Join(strMessages, " "))
}
