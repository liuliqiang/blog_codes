run with following codes without any other third tools:

```
app.run(debug=False, threaded=False, processes=1)
```


This is ApacheBench, Version 2.3 <$Revision: 1757674 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking 127.0.0.1 (be patient).....done


Server Software:        Werkzeug/0.12.2
Server Hostname:        127.0.0.1
Server Port:            5000

Document Path:          /
Document Length:        23 bytes

Concurrency Level:      10
Time taken for tests:   86.690 seconds
Complete requests:      100
Failed requests:        0
Total transferred:      16900 bytes
HTML transferred:       2300 bytes
Requests per second:    1.15 [#/sec] (mean)
Time per request:       8669.015 [ms] (mean)
Time per request:       866.902 [ms] (mean, across all concurrent requests)
Transfer rate:          0.19 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.1      0       0
Processing:   872 8266 1466.3   8668    9107
Waiting:      871 8266 1466.3   8668    9106
Total:        872 8266 1466.3   8668    9107

Percentage of the requests served within a certain time (ms)
  50%   8668
  66%   8807
  75%   8878
  80%   8896
  90%   8947
  95%   9013
  98%   9073
  99%   9107
 100%   9107 (longest request)
