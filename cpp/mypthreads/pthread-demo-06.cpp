#include <ctime>
#include <vector>
#include <iostream>
using namespace std;

#include <unistd.h>
#include <pthread.h>

char* BytesToSize(float Bytes) {
    float tb = 1099511627776;
    float gb = 1073741824;
    float mb = 1048576;
    float kb = 1024;

    char returnSize[256];

    if( Bytes >= tb )
        sprintf(returnSize, "%.2f TB", (float)Bytes/tb);
    else if( Bytes >= gb && Bytes < tb )
        sprintf(returnSize, "%.2f GB", (float)Bytes/gb);
    else if( Bytes >= mb && Bytes < gb )
        sprintf(returnSize, "%.2f MB", (float)Bytes/mb);
    else if( Bytes >= kb && Bytes < mb )
        sprintf(returnSize, "%.2f KB", (float)Bytes/kb);
    else if ( Bytes < kb)
        sprintf(returnSize, "%.2f Bytes", Bytes);
    else
        sprintf(returnSize, "%.2f Bytes", Bytes);

    return returnSize;
}

#define MAX_NUM 10000

int popCount = 0;
vector<int> testVec;

void* pushFunc(void*) {
    for (int i = 0; i < MAX_NUM; i++) {
        testVec.push_back(i);
    }
}

void* popFunc(void*) {
    while (popCount < MAX_NUM) {
        if (!testVec.empty()) {
            testVec.pop_back();
            popCount ++;
        }
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
    cout << "final count = " << popCount << endl;
    return 0;
}