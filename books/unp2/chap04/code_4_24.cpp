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
    int readFifo, writeFifo;
    size_t len;
    ssize_t n;
    char *ptr, fifoName[MAXLINE], buff[MAXLINE];
    pid_t pid;

    pid = getpid();
    snprintf(fifoName, sizeof(fifoName), "/tmp/fifo.%ld", (long)pid);
    if ((mkfifo(fifoName, FILE_MODE) < 0) && (errno != EEXIST)) {
        cerr << "can't create " << fifoName << endl;
        exit(-1);
    }

    snprintf(buff, sizeof(buff), "%ld ", (long)pid);
    len = strlen(buff);
    ptr = buff + len;
    fgets(ptr, MAXLINE - len, stdin);
    len = strlen(buff);

    writeFifo = open(SERV_FIFO, O_WRONLY, 0);
    write(writeFifo, buff, len);

    readFifo = open(fifoName, O_RDONLY, 0);

    while ((n = read(readFifo, buff, MAXLINE)) > 0) {
        write(STDOUT_FILENO, buff, n);
    }

    close(readFifo);
    unlink(fifoName);
    exit(0);
}
