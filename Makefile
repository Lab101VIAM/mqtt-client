
viam-mqtt: *.go 
	go build -o bin/viam-mqtt

lint:
	gofmt -w -s .

module: viam-mqtt
	tar czf module.tar.gz filtered-camera

all: module