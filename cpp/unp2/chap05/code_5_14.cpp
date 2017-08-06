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

int fds[2];
static void sigUsr1(int);

int main(int argc, char* *argv) {
    int nfds;
    char c;
    fd_set rset;
    mqd_t mqd;
    mq_attr attr;
    sigevent sigEv;
    ssize_t n;
    char* buff;

    if (argc != 2) {
        cerr << "usage: " << argv[0] << " <name>" << endl;
        exit(-1);
    }

    mqd = mq_open(argv[1], O_RDONLY);
    mq_getattr(mqd, &attr);
    buff = (char*) malloc(attr.mq_msgsize);

    pipe(fds);

    signal(SIGUSR1, sigUsr1);
    sigEv.sigev_notify = SIGEV_SIGNAL;
    sigEv.sigev_signo = SIGUSR1;
    int notifyResult = mq_notify(mqd, &sigEv);
    if (notifyResult == -1) {
        cerr << "mq_notify error: " << strerror(errno) << endl;
        exit(-2);
    }

    FD_ZERO(&rset);

    while (true) {
        FD_SET(fds[0], &rset);
        nfds = select(fds[0] + 1, &rset, NULL, NULL, NULL);

        if (FD_ISSET(fds[0], &rset)) {
            read(fds[0], &c, 1);
        }

        mq_notify(mqd, &sigEv);

        while ((n = mq_receive(mqd, buff, attr.mq_msgsize, NULL)) >= 0) {
            cout << "read " << n << " bytes " << endl;
        }
        if (errno != EAGAIN) {
            cerr << "mq_receive error: " << strerror(errno) << endl;
            exit(-2);
        }
    }

    exit(0);
}


static void sigUsr1(int sigNo) {
    write(fds[1], "", 1);
    return ;
}
