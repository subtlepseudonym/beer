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
    flow_constant = 0 # scalar
    total_events = 0 # scalar
    total_pour_events = 0 # scalar
    last_event = 0 # milliseconds
    total_pour_time = 0.0 # milliseconds
    total_frequency = 0.0 # 1 / seconds
    total_flow_rate = 0.0 # liters / seconds
    total_pour = 0.0 # liters
    remaining_volume = 0 # liters

    def __init__(self, filepath, flow_constant=THREE_EIGHTHS_CONSTANT):
        if not os.path.isfile(filepath):
            raise ValueError('File "%s" not found' % filepath)

        with open(filepath) as f:
            data = json.load(f)
            self.total_events = data['totalEvents']
            self.total_pour_time = data['totalPourTime']
            self.total_frequency = data['totalFrequency']
            self.total_flow_rate = data['totalFlowRate']
            self.total_pour = data['totalPour']
            self.remaining_volume = data['remainingVolume']

        self.last_event = current_milli_time()
        self.flow_constant = flow_constant

    def to_json(self):
        return { 'totalEvents': self.total_events,
                 'totalPourEvents': self.total_pour_events,
                 'totalPourTime': self.total_pour_time,
                 'totalFrequency': self.total_frequency,
                 'totalFlowRate': self.total_flow_rate,
                 'totalPour': self.total_pour,
                 'remainingVolume': self.remaining_volume,
                 'flowConstant': self.flow_constant }

    def stats(self):
        return { 'avgFreq': self.total_frequency / self.total_events if self.total_events else 0,
                 'avgFlow': self.total_flow_rate / self.total_events if self.total_events else 0,
                 'avgPour': self.total_pour / self.total_pour_events if self.total_pour_events else 0,
                 'totalPour': self.total_pour,
                 'totalPourTime': self.total_pour_time,
                 'totalPourEvents': self.total_pour_events }

    def update(self, now):
        event_delta = max((now - self.last_event), 1)

        if event_delta < DELTA_THRESHOLD:
            self.total_events += 1

            frequency = MS_PER_SECOND / event_delta # ms_per_second / ms_since_last_event
            self.total_frequency += frequency

            flow_rate = frequency / (SECONDS_PER_MINUTE * self.flow_constant) # frequency / (seconds_per_minute * flow_meter_constant)
            self.total_flow_rate += flow_rate

            pour_time = event_delta / MS_PER_SECOND
            self.total_pour_time += pour_time

            pour = flow_rate * pour_time # in liters
            self.total_pour += pour # pour is (1/1260)L per event
            self.remaining_volume -= pour
        else:
            self.total_pour_events += 1

        self.last_event = now

    def event(self, channel):
        now = current_milli_time()
        self.update(now)


flow_meter = FlowMeter('./initial-state.json', 10.5) # constant of 10.5 works better during testing

GPIO.setmode(GPIO.BCM)
GPIO.setup(FLOW_PIN, GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.add_event_detect(FLOW_PIN, GPIO.RISING, callback=flow_meter.event, bouncetime=20)

save_interval = 5
save_counter = 0
while True:
    print("state:", json.dumps(flow_meter.to_json()))
    print("avg:", json.dumps(flow_meter.stats()))

    save_counter += 1
    if save_counter >= save_interval:
        with open('./state.json', 'w') as f:
            f.truncate()
            json.dump(flow_meter.to_json(), f)
        save_counter = 0

    time.sleep(1)

