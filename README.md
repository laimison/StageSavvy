# Stage Savvy

Stage Savvy is a MIDI translator that is capable of translating any MIDI message to any in Ableton. It installs as MIDI Remote Script inside Ableton so no extra app is needed.

## Support

Mac OS

Ableton 12.1

## Installation

1) Download this repository as zip and unpack all files at /Users/your-user/Music/Ableton/User Library/Remote Scripts/StageSavvy

2) Configure Settings.txt , some examples:

```
[MP1]
"ODD Bluetooth.NOTE.0.0.X" = "StageSavvy.NOTE.2.38.X.10ms"
"ODD Bluetooth.NOTE.0.0.50" = "StageSavvy.NOTE.2.39.100.10ms"
"ODD Bluetooth.NOTE.0.0.127" = "[MP2]"

[MP2]
"ODD Bluetooth.NOTE.0.0.1" = "StageSavvy.CC.2.100.1"
"ODD Bluetooth.NOTE.0.0.127" = "[MP3]"

[MP3]
"ODD Bluetooth.CC.0.4.X" = "StageSavvy.CC.2.100.X"
"ODD Bluetooth.NOTE.0.0.127" = "[MP1]"
```

Explanations:

If I receive NOTE0 message on channel 1 from your-midi-device, I translate it to NOTE38 message on channel 3, where X is modifier (eg 1 is 1, 50 is 50, etc).

If I receive NOTE0 message on channel 1 from your-midi-device and velocity is 50, I explicitly translate it to NOTE39 message on channel 3 and velocity 100.

If I receive NOTE0 message on channel 1 from your-midi-device, I translate it to CC100 message on channel 3.

If I receive CC4 message on channel 1 from your-midi-device, I translate it to CC100 on channel 3 where X is modifier (value 5 is value 5, value 50 is value 50 translated)

Additionally, I change preset to next if I receive NOTE2 on channel 1 at velocity 127 from your-midi-device. So it is possible to change the mapping on the fly.

3) Start Ableton and setup as per screenshot

![Configuration](images/Configuration.png)

So Ableton listens only to StageSavvy MIDI device and ignores the other MIDI device

## App Info

A Program mainly runs in a separate Golang thread due to latency reasons. It doesn't depend on a Python processing, Ableton MIDI Remote Script base. Therefore, this program should take only a millisecond or a few to translate the message. So experience to play live is satisfied.

## Troubleshooting

Check log file /Users/your-user/Library/Preferences/Ableton/Live-version-here/Log.txt

You can open Github issue by sending your log

Thanks