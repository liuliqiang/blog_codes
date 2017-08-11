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

volatile sig_atomic_t mqFlag;
static void sigUsr1(int);

int main(int argc, char* *argv) {
    mqd_t mqd;
    mq_attr attr;
    sigevent sigEv;
    sigset_t zeromask, newmask, oldmask;
    ssize_t n;
    char* buff;

    if (argc != 2) {
        cerr << "usage: " << argv[0] << " <name>" << endl;
        exit(-1);
    }

    mqd = mq_open(argv[1], O_RDONLY);
    mq_getattr(mqd, &attr);
    buff = (char*) malloc(attr.mq_msgsize);

    sigemptyset(&zeromask);
    sigemptyset(&newmask);
    sigemptyset(&oldmask);

    signal(SIGUSR1, sigUsr1);
    sigEv.sigev_notify = SIGEV_SIGNAL;
    sigEv.sigev_signo = SIGUSR1;
    int notifyResult = mq_notify(mqd, &sigEv);
    if (notifyResult == -1) {
        cerr << "mq_notify error: " << strerror(errno) << endl;
        exit(-2);
    }

    while (true) {
        sigprocmask(SIG_BLOCK, &newmask, &oldmask);

        while (mqFlag == 0) {
            sigsuspend(&zeromask);
        }
        mqFlag = 0;

        mq_notify(mqd, &sigEv);
        n = mq_receive(mqd, buff, attr.mq_msgsize, NULL);
        cout << "read " << n << " bytes " << endl;
        sigprocmask(SIG_UNBLOCK, &newmask, NULL);
    }

    exit(0);
}


static void sigUsr1(int sigNo) {
    mqFlag = 1;
    return ;
}
