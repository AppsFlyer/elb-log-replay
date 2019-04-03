run:
	go run main.go play \
		--log-files=/Users/ran/Downloads/ \
		--target-host http://localhost:8080 \
		--rate 1000

build-linux:
	GOOS=linux GOARCH=386 go build
