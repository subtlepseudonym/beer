## Beer
This project utilizes a raspberry pi zero w and a flow meter to monitor keg levels 

### Dependency modifications
- github.com/d2r2/go-dht
	- logger.go: logging level changed to Error
	- gpio.h: uncommented sleep between /sys/class/gpio/export write and /sys/class/gpio/gpioX/direction write
- udev rules on raspi
	- SUBSYSTEM="gpio", ACTION="add", PROGRAM="/bin/sh -c 'chgrp -R gpio /sys%p && chmod -R 770 /sys%p'"
