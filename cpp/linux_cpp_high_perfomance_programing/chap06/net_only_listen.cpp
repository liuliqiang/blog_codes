//
// Created by yetship on 2017/7/3.
//
#include <iostream>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <sys/socket.h>
#include <netdb.h>
#include <unistd.h>
#include <cstdlib>

using namespace std;


int main() {
    int servFd = socket(PF_INET, SOCK_STREAM, 0);
    sockaddr_in servAddr;

    servAddr.sin_family = AF_INET;
    inet_pton(AF_INET, "127.0.0.1", &servAddr.sin_addr);
    servAddr.sin_port = htons(3001);

    bind(servFd, (const sockaddr*)&servAddr, sizeof(servAddr));
    listen(servFd, 5);

    while (true) {
        sleep(1);
    }

    close(servFd);
    return 0;
}
