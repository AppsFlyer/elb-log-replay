run:
	go run main.go play \
		--log-file=/Users/ran/Downloads/195229424603_elasticloadbalancing_eu-west-1_clicks_20190327T0000Z_52.17.28.76_4ckkfc6c.txt \
		--target-host http://localhost:8080 \
		--rate 1000000 
