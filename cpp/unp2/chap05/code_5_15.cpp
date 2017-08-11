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
mq_attr attr;
sigevent sigEv;

static void notifyThread(union sigval);

int main(int argc, char* *argv) {
    if (argc != 2) {
        cerr << "usage: " << argv[0] << " <name>" << endl;
        exit(-1);
    }

    mqd = mq_open(argv[1], O_RDONLY | O_NONBLOCK);
    mq_getattr(mqd, &attr);

    sigEv.sigev_notify = SIGEV_THREAD;
    sigEv.sigev_value.sival_ptr = NULL;
    sigEv.sigev_notify_function = notifyThread;
    sigEv.sigev_notify_attributes = NULL;

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


static void notifyThread(union sigval arg) {
    ssize_t n;
    char *buff;

    cout << "notify_thread started" << endl;
    buff = (char*) malloc(attr.mq_msgsize);
    mq_notify(mqd, &sigEv);

    while ((n = mq_receive(mqd, buff, attr.mq_msgsize, NULL)) >= 0) {
        cout << "read " << n << " bytes " << endl;
    }

    if (errno != EAGAIN) {
        cerr << "mq_receive error: " << strerror(errno) << endl;
    }

    free(buff);
    pthread_exit(NULL);
}
