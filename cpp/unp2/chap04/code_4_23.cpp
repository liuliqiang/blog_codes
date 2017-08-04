//
// Created by yetship on 2017/8/4.
//
#include <cstdio>
#include <iostream>
#include <errno.h>
#include <fcntl.h>
using namespace std;

#include "apue.h"

#define SERV_FIFO   "/tmp/fifo.serv"


static ssize_t my_read(int fd, char *ptr)
{
    static int	read_cnt = 0;
    static char	*read_ptr;
    static char	read_buf[MAXLINE];

    if (read_cnt <= 0) {
        again:
        if ( (read_cnt = read(fd, read_buf, sizeof(read_buf))) < 0) {
            if (errno == EINTR)
                goto again;
            return(-1);
        } else if (read_cnt == 0)
            return(0);
        read_ptr = read_buf;
    }

    read_cnt--;
    *ptr = *read_ptr++;
    return(1);
}

ssize_t readline(int fd, void *vptr, size_t maxlen)
{
    int		n, rc;
    char	c, *ptr;

    ptr = (char*)vptr;
    for (n = 1; n < maxlen; n++) {
        if ( (rc = my_read(fd, &c)) == 1) {
            *ptr++ = c;
            if (c == '\n')
                break;	/* newline is stored, like fgets() */
        } else if (rc == 0) {
            if (n == 1)
                return(0);	/* EOF, no data read */
            else
                break;		/* EOF, some data was read */
        } else
            return(-1);		/* error, errno set by read() */
    }

    *ptr = 0;	/* null terminate like fgets() */
    return(n);
}


int main() {
    int readFifo, writeFifo, dummyFd, fd;
    char *ptr, buff[MAXLINE+1], fifoName[MAXLINE];
    pid_t pid;
    ssize_t n;

    if ((mkfifo(SERV_FIFO, FILE_MODE) < 0) && (errno != EEXIST)) {
        cerr << "can't create " << SERV_FIFO << endl;
        exit(-1);
    }

    readFifo = open(SERV_FIFO, O_RDONLY, 0);
    dummyFd = open(SERV_FIFO, O_WRONLY, 0);

    while ((n = readline(readFifo, buff, MAXLINE)) > 0) {
        if (buff[n - 1] == '\n') {
            n --;
        }

        buff[n] = '\0';

        if ((ptr = strchr(buff, ' ')) == NULL) {
            cerr << "bogus request: " << buff << endl;
            continue;
        }

        *ptr++ = 0;

        pid = atol(buff);
        snprintf(fifoName, sizeof(fifoName), "/tmp/fifo.%ld", (long)pid);
        if ((writeFifo = open(fifoName, O_WRONLY, 0)) < 0) {
            cerr << "cannot open " << fifoName << endl;
            continue;
        }

        if ((fd = open(ptr, O_RDONLY)) < 0) {
            snprintf(buff + n, sizeof(buff) - n, ": can't open, %s\n", strerror(errno));
            n = strlen(ptr);
            write(writeFifo, ptr, n);
            close(writeFifo);
        } else {
            while ((n = read(fd, buff, MAXLINE)) > 0) {
                write(writeFifo, buff, n);
            }
            close(fd);
            close(writeFifo);
        }
    }

    exit(0);
}

