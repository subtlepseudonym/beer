import os.path
import time
import json
import RPi.GPIO as GPIO

FLOW_PIN = 14
DELTA_THRESHOLD = 1000 # in milliseconds
THREE_EIGHTHS_CONSTANT = 21 # flow constant determined by manufacturer
ONE_FOURTH_CONSTANT = 38

MS_PER_SECOND = 1000
SECONDS_PER_MINUTE = 60
PINTS_PER_LITER = 2.1133764188651872
GAL_PER_LITER = 0.2641720523581484

current_milli_time = lambda: int(time.time() * MS_PER_SECOND)

class FlowMeter:
    totalEvents = 0
    lastEvent = 0 # in milliseconds
    totalFreq = 0.0 # in 1/ms
    totalFlow = 0.0 # in liters per second
    totalPour = 0.0 # in liters
    remainingVolume = 0 # will be initialized by json file

    def __init__(self, filepath):
        if not os.path.isfile(filepath):
            raise ValueError('File "%s" not found' % filepath)

        with open(filepath) as f:
            data = json.load(f)
            self.totalEvents = data['totalEvents']
            self.totalFreq = data['totalFreq']
            self.totalFlow = data['totalFlow']
            self.totalPour = data['totalPour']
            self.remainingVolume = data['remainingVolume']

        self.lastEvent = current_milli_time()

    def update(self, now):
        eventDelta = max((now - self.lastEvent), 1)

        if eventDelta < DELTA_THRESHOLD:
            self.totalEvents += 1

            hz = MS_PER_SECOND / eventDelta # ms_per_second / ms_since_last_event
            self.totalFreq += hz

            flow = hz / (SECONDS_PER_MINUTE * 21) # frequency / (seconds_per_minute * flow_meter_constant)
            self.totalFlow += flow

            pour = flow * (eventDelta / MS_PER_SECOND) # in liters
            self.totalPour += pour # pour is (1/1260)L per event

        self.lastEvent = now

    def flowEvent(self, channel):
        now = current_milli_time()
        flowMeter.update(now)


flowMeter = FlowMeter('./state.json')

GPIO.setmode(GPIO.BCM)
GPIO.setup(FLOW_PIN, GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.add_event_detect(FLOW_PIN, GPIO.RISING, callback=flowMeter.flowEvent, bouncetime=20)

while True:
    print(flowMeter.totalPour)
    time.sleep(1)
