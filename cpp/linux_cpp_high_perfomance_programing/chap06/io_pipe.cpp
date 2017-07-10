//
// Created by yetship on 2017/7/10.
//

#include <iostream>
#include <unistd.h>
using namespace std;


int main() {
    int fd[2];

    if (-1 == pipe(fd)) {
        cout << "pipe error" << endl;
        return -1;
    }

    pid_t pid = fork();
    if (pid < 0) {
        cout << "fork error" << endl;
    } else if (pid == 0) {
        // child
        close(fd[0]);
        char content[] = "世界和平";
        write(fd[1], content, sizeof(content));
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