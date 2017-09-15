#include <ctime>
#include <cstdio>
#include <cstdlib>
#include <vector>
#include <iostream>

using namespace std;

#include <unistd.h>
#include <pthread.h>

#define MAX_NUM 10000

int numList[MAX_NUM];
int readIdx = 0;
int writeIdx = 0;
int readCount = 0;

pthread_cond_t count_cond = PTHREAD_COND_INITIALIZER;
pthread_mutex_t count_mutex = PTHREAD_MUTEX_INITIALIZER;

void* pushFunc(void*) {
    for (; writeIdx < MAX_NUM; writeIdx++) {
        numList[writeIdx] = writeIdx;
        pthread_cond_signal(&count_cond);
    }
}

void* popFunc(void*) {
    while (readIdx < writeIdx || writeIdx == 0) {
        pthread_mutex_lock(&count_mutex);
        if (readIdx < writeIdx) {
            numList[readIdx] = -1;
            readIdx ++;
            readCount ++;
        } else if (readCount == MAX_NUM) {
            // notice other thread to break
            pthread_cond_signal(&count_cond);
        }
        pthread_mutex_unlock(&count_mutex);
    }
}

int main()
{
    pthread_t threadIds[4];

    int ret = pthread_create(&threadIds[0], NULL, pushFunc, NULL);
    if (ret) {
        perror("create push thread error");
        exit(-1);
    }
    for (int i = 1; i < 4; i++) {
        ret = pthread_create(&threadIds[i], NULL, popFunc, NULL);
        if (ret) {
            perror("create pop thread error");
            exit(-i);
        }
    }
    for (int i = 0; i < 4; i++) {
        pthread_join(threadIds[i], NULL);
    }
    cout << "final count = " << readCount << endl;
    return 0;
}