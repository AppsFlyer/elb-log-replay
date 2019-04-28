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
$ ./elb-log-replay play --log-files=...
```

### CLI args

`--log-files` - Path to log files. Log files end with `*.txt` or `*.log`. All files in the log-files dir ending with txt will be considered as log files and played in alphabetical order.  
`--target-host` - The target host to replay this traffic to. Could either start with `http` or `https`  
`--rate` - The maximum rate per second to replay the log files. An integer number between 1 and infinity
`--num-senders` - number of concurrent HTTP executors (senders) that play traffic. Tune both `rate` and `num-senders` for optimal performance.



## Log file format support

We currently support only ELB classic log format. Although it's easy to extend to other formats (newwer ELB log formats) if demand exist. 

## Design and implementation
One or the most important design goals of this tool is to be able to provide the highest throughput. The other important goal is reliability and accuracy or measurement.  
There are two importnat factors that need to be taken into consideration, the `rate` and the `num-senders`. 
* `rate` describes the maximum send rate, in other words the *desiarable* throughput. 
* `num-senders` describes the number of concurrent HTTP requests that are allowed. 

These two numbers are correlated, for example if rate (desired throughput) is 1000 per second and num-senders is 10 then we can cope with an average request latency of up to 100ms. If the average request latency is > 100ms then we won't be able to keep up with the desired rate. In order to keep up with the desired rate we must increase the number of senders.  

What seems like a simple solution: increasing the number of senders to the max - isn't always a good idea. For example if you set the number of senders to 1k, 10k or 100k or even 1M (that's OK, Golang can still handle these numbers of goroutines), what you'd find is that if your server is slow then request start to time out. Therefore a fine equilibrium must be maintained b/w the request latency (determined by the network and server, not you), rate (determined by you) and num-senders (also determined by you). 

To make this possible we chose a design pattern called **pipelines** also described here https://blog.golang.org/pipelines

1. A single routine reads the log files and sends them one by one to a channel.  
1. On the other side of this channel there are `num-senders` routines that consume from that channel and for each log line they: parse the log line and synchronously send the request and wait for the response.

Eventually we collect the stats (actual rate and latency) and output them for monitoring puprposes. 

```
                                                                 +------+ send to server
+------+                                                 +------>|sender|--------------->
| Log  +--------------+                                  |       +------+
| file |              |                                  |
+------+              v                                  |       +------+ send to server
                 +----------+                            | ----->|sender|--------------->
+------+         |  Log     |                 +----------+--+    +------+
| Log  |         |  files   +---------------->|    channel  |
| file +-------->|  reader  |  rate limited   +----------+--+    +------+ send to server
+------+         +----------+                            | ----->|sender|--------------->
                      ^                                  |       +------+
+------+              |                                  |
| Log  |              |                                  |       +------+ send to server
| file +--------------+                                  +------>|sender|--------------->
+------+                                                         +------+

```

## Benchmark
Throughput is important so here's a small scale benchmark resutls.  TLDR: We're able to play logs 60k lines per second

Hardware: 
* elb-log-replay run on AWS `m5.2xlarge` instance
* Sending request to a local network nginx server running on another AWS `m5.xlarge` instance

Throughput: **`60k`** reqests per second . 
Latency: submilisecond (< `1ms`)
