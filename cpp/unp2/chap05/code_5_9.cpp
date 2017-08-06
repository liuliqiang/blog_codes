#include <iostream>
#include <cstdlib>
#include <cstdint>

#include <fcntl.h>
#include <errno.h>
#include <unistd.h>
#include <mqueue.h>

using namespace std;

#include "apue.h"

#define uint_t uint32_t


mqd_t mqd;
char *buff;
mq_attr attr;
sigevent sigEv;

static void sigUsr1(int);

int main(int argc, char* *argv) {
    if (argc != 2) {
        cerr << "usage: " << argv[0] << " <name>" << endl;
        exit(-1);
    }

    mqd = mq_open(argv[1], O_RDONLY);
    mq_getattr(mqd, &attr);
    buff = (char*) malloc(attr.mq_msgsize);

    signal(SIGUSR1, sigUsr1);
    sigEv.sigev_notify = SIGEV_SIGNAL;
    sigEv.sigev_signo = SIGUSR1;
    int notifyResult = mq_notify(mqd, &sigEv);
    if (notifyResult == -1) {
        cerr << "mq_notify error: " << strerror(errno) << endl;
        exit(-2);
    }

    while (true) {
        pause();
    }

    exit(0);
}


static void sigUsr1(int sigNo) {
    ssize_t n;
    mq_notify(mqd, &sigEv);
    n = mq_receive(mqd, buff, attr.mq_msgsize, NULL);
    cout << "SIGUSR1 received, read " << n << " bytes" << endl;
    return ;
}
