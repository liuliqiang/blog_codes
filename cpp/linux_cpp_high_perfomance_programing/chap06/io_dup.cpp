//
// Created by yetship on 2017/7/10.
//

#include <iostream>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
using namespace std;


int main() {
    int fd[2];

    if (-1 == socketpair(AF_UNIX, SOCK_STREAM, 0, fd)) {
        cout << "pipe error" << endl;
        return -1;
    }

    pid_t pid = fork();
    if (pid < 0) {
        cout << "fork error" << endl;
    } else if (pid == 0) {
        // child
        close(fd[0]);
        char content[] = "世界和平 from dup fd";
        close(STDOUT_FILENO);
        dup(fd[1]);
        cout << content << endl;
        close(fd[1]);
        return -1;
    } else {
        // parent
        close(fd[1]);
        char recvBuf[1024];
        read(fd[0], recvBuf, sizeof(recvBuf));
        cout << "Received from child " << recvBuf << endl;
        return -1;
    }
}