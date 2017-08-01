#include "apue.h"
#include <fcntl.h>
#include <unistd.h>

#include <iostream>

int main()
{
    int fd = open("/tmp/iotest01.txt", O_RDWR | O_CREAT);
    char content[] = "Hello world";
    char buff[MAXLINE];

    write(fd, content, strlen(content));

    off_t seekResult = lseek(fd, 0, SEEK_SET);
    std::cout << "lseek result: " << seekResult << std::endl;
    if (seekResult == -1) {
        err_sys("lseek error");
    }

    std::cout << pread(fd, buff, 5, 0) << std::endl;
    std::cout << "pread: " << buff << std::endl;

    read(fd, buff, strlen(content));
    std::cout << "read: " << buff << std::endl;

    return 0;
}