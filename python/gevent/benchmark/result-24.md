flask multithread

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


Server Software:        Werkzeug/0.12.2
Server Hostname:        127.0.0.1
Server Port:            5000

Document Path:          /
Document Length:        31 bytes

Concurrency Level:      100
Time taken for tests:   4.015 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      177000 bytes
HTML transferred:       31000 bytes
Requests per second:    249.08 [#/sec] (mean)
Time per request:       401.474 [ms] (mean)
Time per request:       4.015 [ms] (mean, across all concurrent requests)
Transfer rate:          43.05 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    1   1.3      0       6
Processing:    47  382  61.3    396     483
Waiting:       41  379  61.2    393     479
Total:         52  383  60.1    397     483

Percentage of the requests served within a certain time (ms)
  50%    397
  66%    404
  75%    408
  80%    411
  90%    423
  95%    435
  98%    451
  99%    465
 100%    483 (longest request)
