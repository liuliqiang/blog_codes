#include <ctime>
#include <iostream>
using namespace std;

#include <unistd.h>
#include <pthread.h>


#define MAX_BUFF 1024

int buffLen = 0;
char buff[MAX_BUFF];


void* sendMsgFunc(void* ptr) {
    char* chrPtr = (char*)ptr;
    int chrLen = strlen(chrPtr);
    if (chrLen >= MAX_BUFF) {
        perror("str too long");
        exit(-1);  // str too long
    }

    memcpy(buff, ptr, chrLen + 1);
    buffLen = chrLen;
}

void* recvMsgFunc(void* ptr) {
    while (buffLen== 0) {
        sleep(1);  // sleep one second
    }
    cout << "recv msg len: " << buffLen << endl;
    cout << "msg content: " << buff << endl;
}

int main()
{
    pthread_t thread01, thread02;

    int ret = pthread_create(&thread01, NULL, recvMsgFunc, NULL);
    if (ret) {
        perror("create thread recvMsgFunc error");
        exit(-2);
    }

    ret = pthread_create(&thread02, NULL, sendMsgFunc, (void*)"Hello World");
    if (ret) {
        perror("create thread sendMsgFunc error");
        exit(-3);
    }

    pthread_join(thread01, NULL);
    pthread_join(thread02, NULL);

    return 0;
}