## Kegerator

[![github](https://img.shields.io/github/v/tag/subtlepseudonym/kegerator?logo=github&sort=semver)](https://github.com/subtlepseudonym/kegerator/tags) [![kofi](https://img.shields.io/badge/ko--fi-Support%20me%20-hotpink?logo=kofi&logoColor=white)](https://ko-fi.com/subtlepseudonym)

This project utilizes a raspberry pi zero w and a flow meter to monitor keg levels 

### Dependency modifications
This is a list of dependency modifications to help this project run a bit better. Relevant entries will be removed if the project moves to vendored dependencies.

- github.com/d2r2/go-dht
	- logger.go: logging level changed to Error
	- gpio.h: uncommented sleep between /sys/class/gpio/export write and /sys/class/gpio/gpioX/direction write
- udev rules on raspi
	- SUBSYSTEM="gpio", ACTION="add", PROGRAM="/bin/sh -c 'chgrp -R gpio /sys%p && chmod -R 770 /sys%p'"
