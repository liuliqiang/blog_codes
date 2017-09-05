run with following codes with gevent WSGIServer

```
http_server = WSGIServer(('', 5000), app)
```


This is ApacheBench, Version 2.3 <$Revision: 1757674 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking 127.0.0.1 (be patient).....done


Server Software:
Server Hostname:        127.0.0.1
Server Port:            5000

Document Path:          /
Document Length:        23 bytes

Concurrency Level:      10
Time taken for tests:   85.935 seconds
Complete requests:      100
Failed requests:        0
Total transferred:      15000 bytes
HTML transferred:       2300 bytes
Requests per second:    1.16 [#/sec] (mean)
Time per request:       8593.544 [ms] (mean)
Time per request:       859.354 [ms] (mean, across all concurrent requests)
Transfer rate:          0.17 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.1      0       1
Processing:   894 8178 1406.2   8549    9117
Waiting:      893 8178 1406.2   8549    9117
Total:        894 8178 1406.2   8549    9117

Percentage of the requests served within a certain time (ms)
  50%   8549
  66%   8610
  75%   8705
  80%   8808
  90%   8910
  95%   8946
  98%   9113
  99%   9117
 100%   9117 (longest request)
