#include <iostream>
#include <cstdlib>
#include <cstdint>

#include <fcntl.h>
#include <errno.h>

#include <mqueue.h>

using namespace std;

#include "apue.h"

#define uint_t uint32_t


int main(int argc, char* *argv) {
    mqd_t mqd;
    char *ptr;
    size_t len;
    uint_t prio;

    if (argc != 4) {
        cerr << "usage: " << argv[0] << " <name> <#bytes> <priority>" << endl;
        exit(-1);
    }

    len = atoi(argv[2]);
    prio = atoi(argv[3]);

    mqd = mq_open(argv[1], O_WRONLY);

    ptr = (char*)calloc(len, sizeof(char));
    int sendResult = mq_send(mqd, ptr, len, prio);
    if (sendResult == -1) {
        cerr << "mq_send error: " << strerror(errno) << endl;
        exit(-2);
    }

    exit(0);
}
