//
// Created by yetship on 2017/7/10.
//

#include <iostream>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/mman.h>
using namespace std;

#define MEMORY_SIZE 1024

void failProcess(int fd, string info) {
    close(fd);
    perror(info.c_str());
    exit(EXIT_FAILURE);
}

int main() {
    int fd = open("/tmp/test.tmp", O_RDWR | O_CREAT | O_TRUNC);

    // 重要的一步
    int result = lseek(fd, MEMORY_SIZE-1, SEEK_SET);
    if (result == -1) {
        failProcess(fd, "Error calling lseek() to 'stretch' the file");
    }
    // 重要的第二步
    result = write(fd, "", 1);
    if (result != 1) {
        failProcess(fd, "Error writing last byte of the file");
    }

    char* mmapAddr = (char*)mmap(NULL, MEMORY_SIZE, PROT_READ | PROT_WRITE,
                                 MAP_SHARED, fd, 0);
    if (mmapAddr == MAP_FAILED) {
        close(fd);
        perror("Error mmapping the file");
        exit(EXIT_FAILURE);
    }

    pid_t pid = fork();
    if (pid < 0) {
        cout << "fork error" << endl;
    } else if (pid == 0) {
        // child
        char content[] = "世界和平 from dup fd";
        cout << "child send content" << endl;
        memcpy(mmapAddr, content, strlen(content) + 1);
        cout << "child send condtent success" << endl;
        return -1;
    } else {
        // parent
        while (mmapAddr[0] == 0) {
            sleep(1);
        }
        char recvBuf[1024];
        memcpy(recvBuf, mmapAddr, strlen(mmapAddr) + 1);
        cout << "Received from child: " << recvBuf << endl;

        for (int idx = 0; idx < 5; idx++)
            sleep(1);
        return -1;
    }
}