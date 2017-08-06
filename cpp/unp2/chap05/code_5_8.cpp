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


int main(int argc, char* *argv) {
    cout << "MQ_OPEN_MAX = " << sysconf(_SC_MQ_OPEN_MAX) << endl;
    cout << "MQ_PRIO_MAX = " << sysconf(_SC_MQ_PRIO_MAX) << endl;

    exit(0);
}
