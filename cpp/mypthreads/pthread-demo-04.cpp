#include <ctime>
#include <iostream>
using namespace std;

#include <unistd.h>
#include <pthread.h>

/**
 * 这个例子将会对超过 BUFF size 的情况做特殊处理
 */


#define MAX_BUFF 10

int buffLen = 0;
char buff[MAX_BUFF];


void* sendMsgFunc(void* ptr) {
    pthread_t threadId = pthread_self();
    cout << "I am thread: " << threadId << endl;

    char* chrPtr = (char*)ptr;
    int chrLen = strlen(chrPtr);
    if (chrLen >= MAX_BUFF) {
        buffLen = -1;
        pthread_exit((void*)"str too long");
    }

    memcpy(buff, ptr, chrLen + 1);
    buffLen = chrLen;
}

void* recvMsgFunc(void* ptr) {
    pthread_t threadId = pthread_self(),
              *waitThreadId = (pthread_t*)ptr;
    cout << "I am in thread: " << threadId << endl;
    pthread_join(*waitThreadId, NULL);

    cout << "recv msg len: " << buffLen << endl;
    if (buffLen > 0) {
        cout << "msg content: " << buff << endl;
    } else {
        cout << "recv msg error" << endl;
    }
}

int main()
{
    pthread_t thread01, thread02;

    int ret = pthread_create(&thread02, NULL, sendMsgFunc, (void*)"Hello World");
    if (ret) {
        perror("create thread sendMsgFunc error");
        exit(-3);
    }
    cout << "create thread: " << thread02 << " success" << endl;

    ret = pthread_create(&thread01, NULL, recvMsgFunc, &thread02);
    if (ret) {
        perror("create thread recvMsgFunc error");
        exit(-2);
    }
    cout << "create thread: " << thread01 << " success" << endl;

    // pthread_join(thread01, NULL);
    pthread_join(thread02, NULL);

    return 0;
}