import time
import RPi.GPIO as GPIO

FLOW_PIN = 14
DELTA_THRESHOLD = 1000 # in milliseconds

MS_PER_SECOND = 1000
SECONDS_PER_MINUTE = 60
PINTS_PER_LITER = 2.11338
GAL_PER_LITER = 0.26417

current_milli_time = lambda: int(time.time() * MS_PER_SECOND)

class FlowMeter():
    events = 0
    lastEvent = 0 # in milliseconds
    totalPour = 0.0 # in liters

    def __init__(self):
        self.events = 0
        self.lastEvent = current_milli_time()
        self.totalPour = 0.0

    def update(self, now):
        print("delta ", eventDelta)

        if eventDelta < DELTA_THRESHOLD:
            hz = MS_PER_SECOND / eventDelta # ms_per_second / ms_since_last_event
            flow = hz / (SECONDS_PER_MINUTE * 7.5) # frequency / (seconds_per_minute * flow_meter_constant)
            pour = flow * (eventDelta / MS_PER_SECOND) # in gallons
            self.totalPour += pour / GAL_PER_LITER

        self.lastEvent = now

    def flowEvent(self, channel):
        now = current_milli_time()
        flowMeter.update(now)

def dummy_callback(channel):
    print("woo")

flowMeter = FlowMeter()

GPIO.setmode(GPIO.BCM)
GPIO.setup(FLOW_PIN, GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.add_event_detect(FLOW_PIN, GPIO.RISING, callback=flowMeter.flowEvent, bouncetime=20)

while True:
    print(flowMeter.totalPour)
    time.sleep(1)
