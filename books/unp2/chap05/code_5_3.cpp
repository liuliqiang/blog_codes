#include <iostream>
#include <cstdlib>

#include <fcntl.h>
#include <errno.h>

#include <mqueue.h>

using namespace std;

#include "apue.h"


int main(int argc, char* *argv) {
    if (argc != 2) {
        cerr << "usage: " << argv[0] << " <name>" << endl;
        exit(-1);
    }

    int result = mq_unlink(argv[1]);

    if (result == -1) {
        cerr << "unlink error: " << strerror(errno) << endl;
        exit(-2);
    }

    exit(0);
}
