#include <iostream>
#include <cstdlib>

#include <fcntl.h>
#include <errno.h>

#include <mqueue.h>

using namespace std;

#include "apue.h"


int main(int argc, char* *argv) {
    int c, flags;
    mqd_t mqd;

    flags = O_RDWR | O_CREAT;
    while ((c = getopt(argc, argv, "e")) != -1) {
        switch(c) {
            case 'e':
                flags |= O_EXCL;
                break;
        }
    }

    if (optind != argc - 1) {
        cerr << "usage: mqcreate [ -e ] <name>" << endl;
        exit(-1);
    }

    mqd = mq_open(argv[optind], flags, FILE_MODE, NULL);
    if (mqd == -1) {
        cerr << "mq_open error: " << strerror(errno) << endl;
        exit(-1);
    }
    mq_close(mqd);
    exit(0);
}
