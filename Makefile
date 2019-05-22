run:
	go run main.go play \
		--log-files=/Users/ran/Downloads/ \
		--target-host http://localhost:8080 \
		--rate 1000 \
		--num-senders 32 \
		--pprof-bind-address :6060

build: get
	go build

get:
	go get

build-linux: get
	GOOS=linux GOARCH=386 go build
