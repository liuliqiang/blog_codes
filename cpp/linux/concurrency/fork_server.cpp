#include <string>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <iostream>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <zconf.h>

using namespace std;

#define HOST "127.0.0.1"
#define PORT 3002
#define QUEUE_SIZE 5
#define BUFFER_SIZE 1024


int main()
{
    sockaddr_in servAddr;

    bzero(&servAddr, sizeof(servAddr));
    servAddr.sin_family = PF_INET;
    inet_pton(AF_INET, HOST, &servAddr.sin_addr);
    servAddr.sin_port = htons(PORT);


    int servFd = socket(PF_INET, SOCK_STREAM, 0);
    if (servFd < 0) {
        perror("create socket error");
        return -1;
    }

    // socket reuse
    int reuse = 1;
    setsockopt(servFd, SOL_SOCKET, SO_REUSEADDR, &reuse, sizeof(reuse));

    int bindRst = bind(servFd, (const struct sockaddr*)&servAddr, (socklen_t)sizeof(servAddr));
    if (bindRst == -1) {
        perror("bind address error");
        return -1;
    }

    // 主动打开
    cout << "open" << endl;
    listen(servFd, QUEUE_SIZE);

    bool stop = false;
    while (!stop) {
        sockaddr_in cliAddr;
        socklen_t cliAddrLen;
        cout << "LISTEN" << endl;
        int clientFd = accept(servFd, (sockaddr*)&cliAddr, &cliAddrLen);
        cout << "ESTABLISHED" << endl;
        if (clientFd) {
            int pid = fork();

            if (pid < 0) {
                perror("fork error");
            } else if (pid == 0) {
                int recvLen;
                char buff[BUFFER_SIZE];
                while ((recvLen = recv(clientFd, (void*) buff, BUFFER_SIZE-1, 0)) != -1) {
                    if (recvLen == 0) {
                        break;
                    }
                    write(clientFd, buff, recvLen);
                }
                close(clientFd);
                close(servFd);
                stop = true;
            }
        }
        close(servFd);
        stop = true;
    }
}