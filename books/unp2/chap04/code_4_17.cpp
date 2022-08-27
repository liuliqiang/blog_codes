/**
 *
 */
#include <cstdio>
#include <iostream>
#include <cstring>
#include <cstdlib>

#include <errno.h>
#include <unistd.h>
#include <fcntl.h>

#include <sys/wait.h>
using namespace std;

#include "apue.h"


#define FIFO1 "/tmp/fifo.1"
#define FIFO2 "/tmp/fifo.2"

void client(int readFd, int writeFd) {
    size_t len;
    ssize_t n;
    char buff[MAXLINE];

    fgets(buff, MAXLINE, stdin);
    len = strlen(buff);
    if (buff[len-1] == '\n') {
        len --;
    }

    write(writeFd, buff, len);

    while ((n = read(readFd, buff, MAXLINE)) > 0) {
        write(STDOUT_FILENO, buff, n);
    }
}

void server(int readFd, int writeFd)
{
    int fd;
    ssize_t n;
    char buff[MAXLINE + 1];

    if ((n = read(readFd, buff, MAXLINE)) == 0)
    {
        cout << "end-of-file while reading pathname" << endl;
        exit(-1);
    }
    buff[n] = '\0';

    if ((fd = open(buff, O_RDONLY)) < 0)
    {
        snprintf(buff + n, sizeof(buff) - n, ": can't open, %s", strerror(errno));
    } else
    {
        while ((n = read(fd, buff, MAXLINE)) > 0)
        {
            write(writeFd, buff, n);
        }
        close(fd);
    }
}


int main() {
    int readFd, writeFd;
    pid_t childPid;

    if ((mkfifo(FIFO1, FILE_MODE) < 0) && (errno != EEXIST)) {
        cerr << "can't create " << FIFO1 << endl;
        exit(-1);
    }
    if ((mkfifo(FIFO2, FILE_MODE) < 0) && (errno != EEXIST)) {
        unlink(FIFO1);
        cerr << "can't create " << FIFO1 << endl;
        exit(-1);
    }

    if ((childPid = fork()) == 0) {
        // child
        readFd = open(FIFO1, O_RDONLY, 0);
        writeFd = open(FIFO2, O_WRONLY, 0);

        server(readFd, writeFd);
        exit(0);
    }
    writeFd = open(FIFO1, O_WRONLY, 0);
    readFd = open(FIFO2, O_RDONLY, 0);

    client(readFd, writeFd);

    waitpid(childPid, NULL, 0);

    close(readFd);
    close(writeFd);

    unlink(FIFO1);
    unlink(FIFO2);
    exit(0);
}
