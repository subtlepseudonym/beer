## Kegerator

[![github](https://img.shields.io/github/v/tag/subtlepseudonym/kegerator?logo=github&sort=semver)](https://github.com/subtlepseudonym/kegerator/tags) [![kofi](https://img.shields.io/badge/ko--fi-Support%20me%20-hotpink?logo=kofi&logoColor=white)](https://ko-fi.com/subtlepseudonym)

This project utilizes a raspberry pi zero w and a flow meter to monitor keg levels 

### Running this project
```bash
docker create \
	--name kegerator \
	--publish 9220:9220 \
	--device /dev/gpiomem \
	--volume "/sys/class/gpio:/sys/class/gpio" \
	--volume "/sys/devices/platform/soc/20200000.gpio/gpiochip0:/sys/devices/platform/soc/20200000.gpio/gpiochip0" \
	--volume "path-to-directory-containing-state.json:/data" \
	subtlepseudonym/kegerator:latest
docker start kegerator
```

The path to `/sys/devices/platform/...` may be incorrect for your system. If this is the case, you can run the following to obtain the correct path:
```bash
echo 1 >> /sys/class/gpio/export
ls -l /sys/class/gpio/gpio1
```

### Known issues
- Permissions for `/sys/class/gpio/gpioX` are not set correctly
	- They should be `root:gpio`, but are `root:root`
	- The workaround for this is included in the command above by mounting `/sys/devices/platform/...`
- Flow meter pins are not correctly detached from
	- For example, using pin 14, `/sys/class/gpio/gpio14` will persist after the container has been stopped
	- Current workarounds:
		- run outside of a docker container
		- run `echo 14 >> /sys/class/gpio/unexport` after stopping the container

### Dependency modifications
This is a list of dependency modifications to help this project run a bit better. Relevant entries will be removed if the project moves to vendored dependencies.

- github.com/d2r2/go-dht
	- logger.go: logging level changed to Error
	- gpio.h: uncommented sleep between /sys/class/gpio/export write and /sys/class/gpio/gpioX/direction write
- udev rules on raspi
	- SUBSYSTEM="gpio", ACTION="add", PROGRAM="/bin/sh -c 'chgrp -R gpio /sys%p && chmod -R 770 /sys%p'"
