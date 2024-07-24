
viam-mqtt: *.go 
	go build -o bin/viam-mqtt

lint:
	gofmt -w -s .

updaterdk:
	go get go.viam.com/rdk@latest
	go mod tidy

module: viam-mqtt
	tar czf module.tar.gz bin/viam-mqtt

all: module