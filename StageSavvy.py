from __future__ import with_statement

import sys
import os
import platform
import Live
import time
import subprocess
import threading
import socket
import re

from _Framework.ControlSurface import ControlSurface # Base class
from _Framework.MidiMap import MidiMap
from _Framework.ControlSurfaceComponent import ControlSurfaceComponent
from _Framework.InputControlElement import MIDI_NOTE_TYPE, MIDI_CC_TYPE

class StageSavvy(ControlSurface):
    __module__ = __name__
    __doc__ = "StageSavvy Script"
    __name__ = "StageSavvy MIDI Remote Script"

    # Define a global variable for sock file
    sock_file = None
    midi_translator_path = None

    def __init__(self, c_instance):
        ControlSurface.__init__(self, c_instance)

        # Initialize global variables
        StageSavvy.sock_file = "/tmp/stagesavvy.sock"
        StageSavvy.midi_translator_path = os.path.realpath(os.path.dirname(os.path.abspath(__file__)) + "/MIDITranslator/MIDITranslator")

        self.log_message('Initialising StageSavvy in Log.txt')
        self.log_message("MIDITranslator's location", StageSavvy.midi_translator_path)

        for n in Live.Application.get_application().control_surfaces:
            if n is not None:
                if type(n) == Live.Application.ControlSurfaceProxy:
                    self.log_message("non legacy control surface detected:", n.type_name)

        # Print environment information
        self.log_message("Using Python version", sys.version)
        self.log_message("Using platform", platform.platform())
        self.log_message("Script's location", os.path.abspath(__file__))

        # Start MIDITranslator
        thread_start_midi_translator = threading.Thread(target=StageSavvy.start_midi_translator, args=(self, StageSavvy.midi_translator_path,))
        thread_start_midi_translator.start()
        
        # Connect via sock file
        thread_connect_socket = threading.Thread(target=StageSavvy.connect_socket, args=(self,))
        thread_connect_socket.start()

        self.c_instance = c_instance

        with self.component_guard():
            self._clip_actions = StageSavvyComponent(self)

    def disconnect(self):
        self.log_message('disconnect')

        # Disconnect sock
        self.sock.close()

        # Stop MIDITranslator
        self.stop_midi_translator()

        ControlSurface.disconnect(self)

    def refresh_state(self):
        self.log_message('refresh_state triggered')
        pass

    def start_midi_translator(self, cmd):
        # Start MIDITranslator
        process = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        while True:
            output = process.stdout.readline()
            if output == b'' and process.poll() is not None:
                break
            if output:
                output_to_print = output.strip().decode("utf-8")
                self.log_message(output_to_print)
        process.wait()

        exit_code = process.returncode

        if exit_code != 0:
            self.log_message("MIDITranslator exited with code", exit_code)

    def stop_midi_translator(self):
        try:
            message = "Stop"
            sock_path = StageSavvy.sock_file
            s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            s.connect(sock_path)
            s.send(message.encode())
            s.close()

            os.system("killall MIDITranslator")
        except:
            pass

    def connect_socket(self):
        time.sleep(0.05)
        try:
            self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            self.sock.connect(StageSavvy.sock_file)
        except:
            pass

    def send_message_via_socket(self, message):
        sock_path = StageSavvy.sock_file
        s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        s.connect(sock_path)
        s.send(message.encode())
        s.close()

    def receive_midi(self, midi_bytes):
        pass

    def build_midi_map(self, midi_map_handle):
        pass

    def _on_track_list_changed(self):
        pass

    def _on_scene_list_changed(self):
        pass

    def _on_clip_triggered(self, clip_slot):
        pass

    def _on_selected_track_changed(self):
        pass

    def _on_selected_scene_changed(self):
        pass

    def _on_selected_clip_changed(self):
        pass
class StageSavvyComponent(ControlSurfaceComponent):
    __module__ = __name__
    __doc__ = "StageSavvy Clip Actions"
    __name__ = "StageSavvy MIDI Remote Script"

    def __init__(self, parent):
        ControlSurfaceComponent.__init__(self)
        self._parent = parent

        parent.log_message('Initialising StageSavvyComponent in Log.txt')

    def disconnect(self):
        self._parent = None

    def set_enabled(self, enabled):
        self._enabled = enabled

    def on_time_changed(self):
        pass

    def on_track_list_changed(self):
        pass

    def on_scene_list_changed(self):
        pass

    def on_selected_track_changed(self):
        pass

    def on_selected_scene_changed(self):
        pass