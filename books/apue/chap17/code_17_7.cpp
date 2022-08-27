#include "apue.h"
#include <sys/socket.h>
#include <sys/un.h>

#include <iostream>

int main(void) {
    int fd, size;
    sockaddr_un un;

    un.sun_family = AF_UNIX;
    strcpy(un.sun_path, "foo.socket");

    if ((fd = socket(AF_UNIX, SOCK_STREAM, 0)) < 0) {
        err_sys("socket failed");
    }
    size = offsetof(sockaddr_un, sun_path) + strlen(un.sun_path);
    if (bind(fd, (sockaddr*)&un, size) < 0) {
        err_sys("bind failed");
    }

    std::cout << "UNIX domain socket bound" << std::endl;
    exit(0);
}