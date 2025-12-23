module github.com/relabs-tech/inertial_computer

go 1.25.5

require (
	github.com/adrianmo/go-nmea v1.10.0
	github.com/eclipse/paho.mqtt.golang v1.5.1
	github.com/gorilla/websocket v1.5.3
	github.com/jacobsa/go-serial v0.0.0-20180131005756-15cf729a72d4
	periph.io/x/conn/v3 v3.7.2
	periph.io/x/devices/v3 v3.7.4
	periph.io/x/host/v3 v3.8.5
)

replace periph.io/x/devices/v3 => /home/dalarub/go/src/github.com/relabs-tech/devices

require (
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
)
