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
    totalPourEvents = 0
    totalPourTime = 0.0 # in milliseconds
    totalFreq = 0.0 # in 1 / seconds
    totalFlow = 0.0 # in liters per second
    totalPour = 0.0 # in liters
    remainingVolume = 0 # will be initialized by json file

    def __init__(self, filepath):
        if not os.path.isfile(filepath):
            raise ValueError('File "%s" not found' % filepath)

        with open(filepath) as f:
            data = json.load(f)
            self.totalEvents = data['totalEvents']
            self.totalPourTime = data['totalPourTime']
            self.totalFreq = data['totalFreq']
            self.totalFlow = data['totalFlow']
            self.totalPour = data['totalPour']
            self.remainingVolume = data['remainingVolume']

        self.lastEvent = current_milli_time()

    def toJSON(self):
        return { 'totalEvents': self.totalEvents,
                 'totalPourEvents': self.totalPourEvents,
                 'totalPourTime': self.totalPourTime,
                 'totalFreq': self.totalFreq,
                 'totalFlow': self.totalFlow,
                 'totalPour': self.totalPour,
                 'remainingVolume': self.remainingVolume }

    def getStats(self):
        return { 'avgFreq': self.totalFreq / self.totalEvents if self.totalEvents else 0,
                 'avgFlow': self.totalFlow / self.totalEvents if self.totalEvents else 0,
                 'avgPour': self.totalPour / self.totalPourEvents if self.totalPourEvents else 0 }

    def update(self, now):
        eventDelta = max((now - self.lastEvent), 1)

        if eventDelta < DELTA_THRESHOLD:
            self.totalEvents += 1

            hz = MS_PER_SECOND / eventDelta # ms_per_second / ms_since_last_event
            self.totalFreq += hz

            flow = hz / (SECONDS_PER_MINUTE * 10.5) # frequency / (seconds_per_minute * flow_meter_constant)
            self.totalFlow += flow

            pourTime = eventDelta / MS_PER_SECOND
            self.totalPourTime += pourTime

            pour = flow * pourTime # in liters
            self.totalPour += pour # pour is (1/1260)L per event
            self.remainingVolume -= pour
        else:
            self.totalPourEvents += 1

        self.lastEvent = now

    def flowEvent(self, channel):
        now = current_milli_time()
        flowMeter.update(now)


flowMeter = FlowMeter('./initial-state.json')

GPIO.setmode(GPIO.BCM)
GPIO.setup(FLOW_PIN, GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.add_event_detect(FLOW_PIN, GPIO.RISING, callback=flowMeter.flowEvent, bouncetime=20)

saveThreshold = 5
saveCount = 0
while True:
    print("state:", json.dumps(flowMeter.toJSON()))
    print("avg:", json.dumps(flowMeter.getStats()))

    saveCount += 1
    if saveCount >= saveThreshold:
        with open('./state.json', 'w') as f:
            f.truncate()
            json.dump(flowMeter.toJSON(), f)
        saveCount = 0

    time.sleep(1)

