LDFLAGS = -lapue -lrt
CFLAGS = -std=c++11
mqcreate: code_5_2.cpp
	g++ code_5_2.cpp -g -o mqcreate ${LDFLAGS} ${CFLAGS}
mqcreate2: code_5_5.cpp
	g++ code_5_5.cpp -g -o mqcreate2 ${LDFLAGS} ${CFLAGS} 
mqunlink: code_5_3.cpp
	g++ code_5_3.cpp -g -o mqunlink ${LDFLAGS}  ${CFLAGS}
mqgetattr: code_5_4.cpp
	g++ code_5_4.cpp -g -o mqgetattr ${LDFLAGS}  ${CFLAGS}
mqsend: code_5_6.cpp
	g++ code_5_6.cpp -g -o mqsend ${LDFLAGS}  ${CFLAGS}
mqreceive: code_5_7.cpp
	g++ code_5_7.cpp -g -o mqreceive ${LDFLAGS}  ${CFLAGS}
mqsysconf: code_5_8.cpp
	g++ code_5_8.cpp -g -o mqsysconf ${LDFLAGS}  ${CFLAGS}
mqnotify: code_5_9.cpp
	g++ code_5_9.cpp -g -o mqnotify ${LDFLAGS}  ${CFLAGS}
mqnotify2: code_5_11.cpp
	g++ code_5_11.cpp -g -o mqnotify2 ${LDFLAGS}  ${CFLAGS}
mqnotify5: code_5_14.cpp
	g++ code_5_14.cpp -g -o mqnotify5 ${LDFLAGS}  ${CFLAGS}
mqnotify6: code_5_15.cpp
	g++ code_5_15.cpp -g -o mqnotify6 ${LDFLAGS}  ${CFLAGS}
clean:
	rm -f mqcreate mqcreate2 mqunlink mqgetattr mqsend mqreceive mqsysconf mqnotify mqnotify2 mqnotify5 mqnotify6
