#include <ctime>
#include <iostream>
using namespace std;

#include <unistd.h>
#include <pthread.h>

/**
 * 这个例子将会将自己的 thread_id 打印出来
 * 我将使用一个比较原始的手段让输出有顺序
 */


#define MAX_BUFF 1024

int buffLen = 0;
char buff[MAX_BUFF];


void* sendMsgFunc(void* ptr) {
    pthread_t threadId = pthread_self();
    cout << "I am thread: " << threadId << endl;

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
    pthread_t threadId = pthread_self(),
              *waitThreadId = (pthread_t*)ptr;
    cout << "I am in thread: " << threadId << endl;
    pthread_join(*waitThreadId, NULL);

    cout << "recv msg len: " << buffLen << endl;
    cout << "msg content: " << buff << endl;
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