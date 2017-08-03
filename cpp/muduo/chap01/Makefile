CC=g++
THIRD_DIR = /Users/luke/github/homebrew/Cellar
CFLAGS = -I${THIRD_DIR}/libevent/2.1.8/include -I${THIRD_DIR}/boost/1.62.0/include
LDFLAGS = -L${THIRD_DIR}/libevent/2.1.8/lib -L${THIRD_DIR}/boost/1.62.0/lib
server: main.cpp
	${CC} main.cpp ${CFLAGS} ${LDFLAGS} -o server -levent -lboost

clean: server
	rm server
