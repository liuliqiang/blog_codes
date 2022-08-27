/**
 * Created by yetship on 2017/8/4.
 * Example for test whether system support full-duplex pipe
 */
#include <cstdlib>
#include <iostream>

#include <unistd.h>
#include <fcntl.h>
#include <sys/wait.h>
using namespace std;

#include "apue.h"


int main() {
    int fd[2], n;
    char c;
    pid_t childPid;

    pipe(fd);
    if ((childPid = fork()) == 0) {
        sleep(3);

        if ((n = read(fd[0], &c, 1)) != 1) {
            cout << "child: read returned: " << n << endl;
            exit(-1);
        }
        cout << "child read: " << c << endl;
        write(fd[0], "c", 1);
        exit(0);
    }
    // parent
    write(fd[1], "p", 1);
    if ((n = read(fd[1], &c, 1)) != 1) {
        cerr << "parent: read returned " << n << endl;
        exit(-1);
    }
    cout << "parent read " << c << endl;
    waitpid(childPid, NULL, 0);
    exit(0);
}
