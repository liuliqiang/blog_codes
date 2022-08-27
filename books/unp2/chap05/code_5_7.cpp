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
    int c, flags;
    char *buff;
    ssize_t n;
    uint_t prio;
    mqd_t mqd;
    mq_attr attr;

    flags = O_RDONLY;
    while ((c = getopt(argc, argv, "n")) != -1) {
        switch (c) {
            case 'n':
                flags |= O_NONBLOCK;
                break;
        }
    }

    if (optind != argc - 1) {
        cerr << "usage: " << argv[0] << " [ -n ] <name>" << endl;
        exit(-1);
    }

    mqd = mq_open(argv[1], flags);
    mq_getattr(mqd, &attr);

    buff = (char*) malloc(attr.mq_msgsize);
    n = mq_receive(mqd, buff, attr.mq_msgsize, &prio);
    cout << "read " << n << " bytes, priority = " << prio << endl;

    exit(0);
}
