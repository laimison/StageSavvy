# from __future__ import with_statement, absolute_import, print_function, unicode_literals
from __future__ import with_statement

import sys
import os
import platform
import Live
import time
import subprocess
import threading
import socket
from tomllib import load

from _Framework.ControlSurface import ControlSurface # Base class
from _Framework.MidiMap import MidiMap
from _Framework.ControlSurfaceComponent import ControlSurfaceComponent
from _Framework.InputControlElement import MIDI_NOTE_TYPE, MIDI_CC_TYPE

class StageSavvy(ControlSurface):
    __module__ = __name__
    __doc__ = "StageSavvy Script"
    __name__ = "StageSavvy MIDI Remote Script"
    
    def __init__(self, c_instance):
        ControlSurface.__init__(self, c_instance)

        self.log_message('Initialising StageSavvy in Log.txt')

        # Print environment information
        self.log_message("Using Python version", sys.version)
        self.log_message("Using platform", platform.platform())
        self.log_message("Script's location", os.path.abspath(__file__))

        # Stop MIDISender
        try:
            sock_path = '/tmp/stagesavvy.sock'
            s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            s.connect(sock_path)
            message = "Stop"
            s.send(message.encode())
            s.close()
        except:
            pass
        
        # Start MIDISender
        cmd = os.path.realpath(os.path.dirname(os.path.abspath(__file__)) + "/MIDISender/MIDISender")
        self.log_message("MIDISender's location", cmd)
        thread1 = threading.Thread(target=StageSavvy.start_midi_sender, args=(self, cmd,))
        thread1.start()
        time.sleep(0.5)

        # Connect socket
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.connect("/tmp/stagesavvy.sock")

        self.c_instance = c_instance

    def performance_test(self):
        # to call this in thread:
        # thread_test = threading.Thread(target=StageSavvy.performance_test, args=(self,))
        # thread_test.start()
        self.log_message("Performance test begins")
        for i in range(10000000):
            i = i + 1
        self.log_message("Perofrmance test ends")

    def start_midi_sender(self, cmd):
        process = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        while True:
            output = process.stdout.readline()
            if output == b'' and process.poll() is not None:
                break
            if output:
                output_to_print = output.strip().decode("utf-8")
                self.log_message(output_to_print)
        process.wait()

    def send_midi_message(self, type, ch, key, value, delay):
        if delay > 0:
            self.log_message("send_midi_message begins with delay")
            time.sleep(int(delay) / 1000)
        else:
            self.log_message("send_midi_message begins without delay")

        message = type + " " + str(ch) + " " + str(key) + " " + str(value)
        self.sock.send(message.encode())

        self.log_message("send_midi_message finished")

    def receive_midi(self, midi_bytes):
        self.log_message("receive_midi begins")

        type = ""
        key = midi_bytes[1]
        value = midi_bytes[2]

        if midi_bytes[0] == 176:
            type = "CC"
        if midi_bytes[0] == 144:
            type = "NoteOn"
        if midi_bytes[0] == 128:
            type = "NoteOff"

        channel = 1

        data = self.print_midi_translation_tree()
        type = type.replace("On", "").replace("Off", "").upper()
        destination = data[type][str(channel)][str(key)][value]

        destination_type = destination[0]
        destination_channel = 1
        destination_key = destination[2]
        destination_value = destination[3]

        destination_length = int(destination[4].replace("ms", ""))

        delay_ms_note_on = 0
        delay_ms_note_off = destination_length
        delay_ms_cc = 0

        if destination_type == "CC":
            self.log_message("receive_midi input " + type + "" + str(key) + " at " + str(value) + ", output " + "CC" + "" + str(destination_key) + " at " + str(destination_value) + " in " + str(delay_ms_cc) + " ms")

            thread_cc = threading.Thread(target=StageSavvy.send_midi_message, args=(self, "CC", destination_channel, destination_key, destination_value, delay_ms_cc))
            thread_cc.start()

        if destination_type == "NOTE":
            self.log_message("receive_midi input " + type + "" + str(key) + " at " + str(value) + ", output " + "NoteOn" + "" + str(destination_key) + " at " + str(destination_value) + " in " + str(delay_ms_note_on) + "ms and do NoteOff in " + str(delay_ms_note_off) + "ms")

            thread_note_on = threading.Thread(target=StageSavvy.send_midi_message, args=(self, "NoteOn", destination_channel, destination_key, destination_value, delay_ms_note_on))
            thread_note_on.start()
            thread_note_off = threading.Thread(target=StageSavvy.send_midi_message, args=(self, "NoteOff", destination_channel, destination_key, 0, delay_ms_note_off))
            thread_note_off.start()

        self.log_message("receive_midi done")

    def update_midi_translation_tree(self, input):
        global midi_translation_tree_global_var
        midi_translation_tree_global_var = input

    def print_midi_translation_tree(self):
        return midi_translation_tree_global_var

    def build_midi_map(self, midi_map_handle):
        self.log_message("build_midi_map")

        script_handle = self.c_instance.handle()

        file = os.path.realpath(os.path.dirname(os.path.abspath(__file__)) + "/Settings.txt")
        with open(file, 'rb') as f:
            settings = load(f)
        self.log_message("Settings.txt:", settings)

        # Variable for MIDI translation settings
        global midi_translation_tree
        midi_translation_tree = {}

        for key, value in settings.items():
            for sub_key, sub_value in value.items():
                type = sub_key.split(".")[0]
                channel = sub_key.split(".")[1]
                key = sub_key.split(".")[2]
                value = sub_key.split(".")[3]

                type_to = sub_value.split(".")[0]
                channel_to = sub_value.split(".")[1]
                key_to = sub_value.split(".")[2]
                value_to = sub_value.split(".")[3]

                if len(sub_value.split(".")) == 5:
                    length_to = sub_value.split(".")[4]
                else:
                    length_to = "0ms"

                if type not in midi_translation_tree:
                    midi_translation_tree[type] = {}

                if channel not in midi_translation_tree[type]:
                    midi_translation_tree[type][channel] = {}

                if key not in midi_translation_tree[type][channel]:
                    midi_translation_tree[type][channel][key] = {}

                if value == "X":
                    for v in range(128):
                        midi_translation_tree[type][channel][key][v] = type_to, channel_to, key_to, v, length_to
                else:
                    midi_translation_tree[type][channel][key][value] = type_to, channel_to, key_to, value_to, length_to

                if type == "CC":
                    self.log_message("Map CC Ch:" + channel + " Key:" + key)
                    Live.MidiMap.forward_midi_cc(script_handle, midi_map_handle, int(channel) - 1, int(key))

                if type == "NOTE":
                    self.log_message("Map Note Ch:" + channel + " Key:" + key)
                    Live.MidiMap.forward_midi_note(script_handle, midi_map_handle, int(channel) - 1 , int(key))

        self.update_midi_translation_tree(midi_translation_tree)

    def refresh_state(self):
        self.log_message('refresh_state')
        pass

    def disconnect(self):
        self.log_message('disconnect')

        # Disconnect sock
        self.sock.close()

        # Stop MIDISender
        try:
            sock_path = '/tmp/stagesavvy.sock'
            s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            s.connect(sock_path)
            message = "Stop"
            s.send(message.encode())
            s.close()
        except:
            pass

        ControlSurface.disconnect(self)