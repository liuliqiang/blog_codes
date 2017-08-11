//
// Created by yetship on 2017/8/4.
//
#include <cstdarg>
#include <cstdio>
#include <cstdlib>
#include <cstring>

#include <errno.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/wait.h>

#include <iostream>
using namespace std;

#include "apue.h"

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

void server(int readFd, int writeFd) {
    int fd;
    ssize_t n;
    char buff[MAXLINE + 1];

    if ((n = read(readFd, buff, MAXLINE)) == 0) {
        cout << "end-of-file while reading pathname" << endl;
        exit(-1);
    }
    buff[n] = '\0';

    if ((fd = open(buff, O_RDONLY)) < 0) {
        snprintf(buff + n, sizeof(buff) - n, ": can't open, %s", strerror(errno));
    } else {
        while ((n = read(fd, buff, MAXLINE)) > 0) {
            write(writeFd, buff, n);
        }
        close(fd);
    }
}

int main() {
    int pipe1[2], pipe2[2];
    pid_t childPid;

    pipe(pipe1);
    pipe(pipe2);


    if ((childPid = fork()) == 0) {
        // child
        close(pipe1[1]);
        close(pipe2[0]);

        server(pipe1[0], pipe2[1]);
        exit(0);
    }
    // parent
    close(pipe1[0]);
    close(pipe2[1]);

    client(pipe2[0], pipe1[1]);
    waitpid(childPid, NULL, 0);
    exit(0);
}
