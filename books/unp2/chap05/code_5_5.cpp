#include <iostream>
#include <cstdlib>

#include <fcntl.h>
#include <errno.h>

#include <mqueue.h>

using namespace std;

#include "apue.h"


mq_attr attr;

int main(int argc, char* *argv) {
    int c, flags;
    mqd_t mqd;

    flags = O_RDWR | O_CREAT;
    while ((c = getopt(argc, argv, "em:z:")) != -1) {
        switch(c) {
            case 'e':
                flags |= O_EXCL;
                break;
            case 'm':
                attr.mq_maxmsg = atol(optarg);
                break;
            case 'z':
                attr.mq_msgsize = atol(optarg);
                break;
        }
    }

    if (optind != argc - 1) {
        cerr << "usage: mqcreate [ -e ]  [ -m maxmsg -z msgsize ] <name>" << endl;
        exit(-1);
    }

    if ((attr.mq_maxmsg != 0 && attr.mq_msgsize == 0) ||
        (attr.mq_maxmsg == 0 && attr.mq_msgsize != 0)) {
        cerr << "must sprcify both -m maxmsg and -z msgsize" << endl;
        exit(-2);
    }

    mqd = mq_open(argv[optind], flags, FILE_MODE, (attr.mq_maxmsg != 0) ? &attr: NULL);
    if (mqd == -1) {
        cerr << "mq_open error: " << strerror(errno) << endl;
        exit(-1);
    }
    mq_close(mqd);
    exit(0);
}
