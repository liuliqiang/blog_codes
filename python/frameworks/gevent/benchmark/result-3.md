run with following codes with gunicorn webserver

``
gunicorn -w 4 -b localhost:5000 app:app
```


This is ApacheBench, Version 2.3 <$Revision: 1757674 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking 127.0.0.1 (be patient).....done


Server Software:        gunicorn/19.7.1
Server Hostname:        127.0.0.1
Server Port:            5000

Document Path:          /
Document Length:        23 bytes

Concurrency Level:      10
Time taken for tests:   42.458 seconds
Complete requests:      100
Failed requests:        0
Total transferred:      17500 bytes
HTML transferred:       2300 bytes
Requests per second:    2.36 [#/sec] (mean)
Time per request:       4245.793 [ms] (mean)
Time per request:       424.579 [ms] (mean, across all concurrent requests)
Transfer rate:          0.40 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.5      0       5
Processing:  1702 4069 800.1   4198    5421
Waiting:     1701 4068 800.3   4197    5420
Total:       1702 4069 800.0   4198    5421

Percentage of the requests served within a certain time (ms)
  50%   4198
  66%   4404
  75%   4463
  80%   4731
  90%   5127
  95%   5219
  98%   5421
  99%   5421
 100%   5421 (longest request)
