#!/usr/bin/env python3

import rtmidi
import time
import datetime
import os
import sys

from rtmidi.midiconstants import CONTROL_CHANGE, NOTE_OFF, NOTE_ON

# Allow 5 arguments
if len(sys.argv) <= 5:
    print("Missing arguments.\n\nUsage examples:\n\n" + sys.argv[0] + " Note 16 1 100 enter\n" + sys.argv[0] + " CC 1 127 127 15s")
    os._exit(1)

# Assignment
type = sys.argv[1]
ch = int(sys.argv[2])
key = int(sys.argv[3])
value = int(sys.argv[4])
wait = sys.argv[5].replace('s', '')

midi_out = rtmidi.MidiOut()
midi_out.open_virtual_port("StageSavvy MIDI Generator")

print("Wait 5 seconds")
time.sleep(5)

try:
    while True:
        current_time = datetime.datetime.now()
        current_time_str = current_time.strftime("%Y-%m-%d %H:%M:%S.%f")
        print(current_time_str + " sending")
    
        if type == "Note":
            midi_out.send_message([NOTE_ON | ch - 1, key, value])

        if type == "CC":
            midi_out.send_message([CONTROL_CHANGE | ch -1, key, value])

        current_time = datetime.datetime.now()
        current_time_str = current_time.strftime("%Y-%m-%d %H:%M:%S.%f")
        print(current_time_str + " sent")

        if type == "Note":
            time.sleep(2)
            midi_out.send_message([NOTE_OFF | ch -1, key, 0])

        if wait == "enter":
            input()
        else:
            time.sleep(int(wait))

except KeyboardInterrupt:
    print("Ctrl+C pressed. Exiting loop.")

del midi_out
