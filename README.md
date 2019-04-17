# elb-log-replay

elb-log-replay is a log replay tool for ELB (AWS Elastic Load Balancer) logs.  
A typical use case when creating a new feature is to download the log files from existing ELBs used by your app and replay them to a staging or a dev server. 

## Usage

Compile and run: 
```
	go run main.go play \
		--log-files=/path/to/log.files/ \
		--target-host http://localhost:8080 \
		--rate 1000 \
		--num-senders 32
```

Or build and run:
```
$ go build
$ elb-log-replay play --log-files=...
```

### CLI args

`--log-files` - Path to log files. Log files end with `*.txt`. All files in the log-files dir ending with txt will be considered as log files and played in alphabetical order.  
`--target-host` - The target host to replay this traffic to. Could either start with `http` or `https`  
`--rate` - The maximum rate per second to replay the log files. An integer number between 1 and infinity
`--num-senders` - number of concurrent HTTP executors (senders) that play traffic. Tune both `rate` and `num-senders` for optimal performance.



## Log file format support

We currently support only ELB classic log format. Although it's easy to extend to other formats (newwer ELB log formats) if demand exist. 


