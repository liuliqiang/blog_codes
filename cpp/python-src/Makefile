INCLUDE=-I/usr/local/include/python3.6m -I/usr/local/include/python3.6m
LOAD=-lpython3.6m -lpthread -ldl  -lutil -lm
test: main.cpp
	g++ -ggdb main.cpp -o test $(INCLUDE) $(LOAD)
clean:
	rm -f test
