#include <ctime>
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

int main()
{
    size_t stackSize;
    pthread_attr_t attr;

    pthread_attr_init(&attr);
    pthread_attr_getstacksize(&attr, &stackSize);

    cout << "default thread stack size = " << BytesToSize(stackSize) << endl;
    return 0;
}