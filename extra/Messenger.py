#!/usr/bin/env python3

import socket
import os
import time
import datetime
import sys

# Sock file
sock_path = '/tmp/stagesavvy.sock'

# Stop MIDISender
if len(sys.argv) > 1 and sys.argv[1] == "Stop":
    s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    s.connect(sock_path)
    s.send("Stop".encode())
    s.close()
    os._exit(0)

# Allow 4 arguments
if len(sys.argv) <= 4:
    print("Missing arguments.\n\nUsage examples:\n\n" + sys.argv[0] + " Note 16 1 100\n" + sys.argv[0] + " CC 1 127 127\n" + sys.argv[0] + " Stop")
    os._exit(1)

# Assignment
type = sys.argv[1]
ch = sys.argv[2]
key = sys.argv[3]
value = sys.argv[4]

try:
    while True:
        current_time = datetime.datetime.now()
        current_time_str = current_time.strftime("%Y-%m-%d %H:%M:%S.%f")
        print(current_time_str + " sending")

        if type == "Note":
            message = "NoteOn " + ch + " " + key + " " + value

            s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            s.connect(sock_path)
            s.send(message.encode())
            s.close()

        if type == "CC":
            message = "CC " + ch + " " + key + " " + value

            s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            s.connect(sock_path)
            s.send(message.encode())
            s.close()

        current_time = datetime.datetime.now()
        current_time_str = current_time.strftime("%Y-%m-%d %H:%M:%S.%f")
        print(current_time_str + " sent")

        if type == "Note":
            time.sleep(2)

            message = "NoteOff " + ch + " " + key + " " + value

            s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            s.connect(sock_path)
            s.send(message.encode())
            s.close()

        time.sleep(5)
except KeyboardInterrupt:
    print("Ctrl+C pressed. Exiting loop.")
