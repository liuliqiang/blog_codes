#include <iostream>
#include <cstdlib>

#include <fcntl.h>
#include <errno.h>

#include <mqueue.h>

using namespace std;

#include "apue.h"


int main(int argc, char* *argv) {
    mqd_t mqd;
    mq_attr attr;

    if (argc != 2) {
        cerr << "usage: " << argv[0] << " <name> " << endl;
        exit(-1);
    }

    mqd = mq_open(argv[1], O_RDONLY);

    mq_getattr(mqd, &attr);
    cout << "max #msgs = " << attr.mq_maxmsg << endl;
    cout << "max #bytes/msg = " << attr.mq_msgsize << endl;
    cout << "max #currently on queue= " << attr.mq_curmsgs << endl;

    mq_close(mqd);
    exit(0);
}
