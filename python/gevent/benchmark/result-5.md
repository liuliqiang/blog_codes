run with following codes with gunicorn and nginx

```
	gunicorn -w 4 -b localhost:5000 app:app
```


This is ApacheBench, Version 2.3 <$Revision: 1757674 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking 127.0.0.1 (be patient)
Completed 100 requests
Completed 200 requests
Completed 300 requests
Completed 400 requests
Completed 500 requests
Completed 600 requests
Completed 700 requests
Completed 800 requests
Completed 900 requests
Completed 1000 requests
Finished 1000 requests


Server Software:        nginx/1.8.0
Server Hostname:        127.0.0.1
Server Port:            8080

Document Path:          /
Document Length:        23 bytes

Concurrency Level:      100
Time taken for tests:   563.064 seconds
Complete requests:      1000
Failed requests:        409
   (Connect: 0, Receive: 0, Length: 409, Exceptions: 0)
Non-2xx responses:      409
Total transferred:      393496 bytes
HTML transferred:       233226 bytes
Requests per second:    1.78 [#/sec] (mean)
Time per request:       56306.420 [ms] (mean)
Time per request:       563.064 [ms] (mean, across all concurrent requests)
Transfer rate:          0.68 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    1   0.9      0       6
Processing:  1527 53351 11982.9  59398   60005
Waiting:     1527 53351 11982.9  59398   60005
Total:       1531 53352 11982.1  59398   60007
WARNING: The median and mean for the initial connection time are not within a normal deviation
        These results are probably not that reliable.

Percentage of the requests served within a certain time (ms)
  50%  59398
  66%  60001
  75%  60002
  80%  60002
  90%  60003
  95%  60003
  98%  60004
  99%  60004
 100%  60007 (longest request)
