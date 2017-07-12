#include <string>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <fcntl.h>
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
#define MAX_CONNECTION 1024
#define DEBUG false


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

    //!!! forget
    fcntl(servFd, F_SETFL, O_NONBLOCK);

    // select
    fd_set inputSet;
    fd_set workingSet;

    int clientNum = 0;
    int clientFds[MAX_CONNECTION];

    FD_ZERO(&inputSet);
    FD_ZERO(&workingSet);

    FD_SET(servFd, &inputSet);

    /**
     * 遗留 BUG，maxFd 怎么减小？
     */
    char buff[BUFFER_SIZE];
    for (int num = 0, maxFd = servFd + 1; true; ){
        // 在使用 FD_SET 的时候不能操作？
        memcpy(&workingSet, &inputSet, sizeof(inputSet));
//        cout << "[.] ready to select" << endl;
        //!!! important need to reset here
        num = select(maxFd, &workingSet, NULL, NULL, NULL);
//        cout << "[.] select return" << endl;
        // server ready
        if (FD_ISSET(servFd, &workingSet)) {
//            cout << "[.] server accept" << endl;
            sockaddr cliAddr;
            socklen_t cliAddrLen;
            bzero(&cliAddr, sizeof(cliAddr));
            int cli = accept(servFd, (sockaddr*)&cliAddr, &cliAddrLen);
            if (cli == -1) {
                perror("accept from server fd error");
                return -1;
            }

            FD_SET(cli, &inputSet);
//            FD_SET(servFd, &inputSet);
            maxFd = cli + 1;
            clientFds[clientNum++] = cli;

            cout << "all fds===" << endl;
            for (int i = 0; i < clientNum; i++) {
                cout << "client: " << clientFds[i] << " exists" << endl;
            }
        } else {
            // client ready
            cout << "client num: " << clientNum << endl;
            for (int i = 0; i < clientNum; i++) {
                int clientSet = FD_ISSET(clientFds[i], &workingSet);
                cout << "client " << clientFds[i] << " set?: " << clientSet << endl;
                if (clientSet) {
                    int recvNum = recv(clientFds[i], (void*) buff, BUFFER_SIZE - 1, 0);
                    if (recvNum > 0) {
                        cout << "recv from client: " << clientFds[i] << ", info: " << buff << endl;
                        send(clientFds[i], buff, strlen(buff) + 1, 0);
                    } else {
                        if (recvNum == 0) {
                            cout << "[x] client: " << clientFds[i] << " closed" << endl;
                        } else {
                            perror("client error");
                        }
                        // !!! important: need to clear here
                        FD_CLR(clientFds[i], &inputSet);
                        close(clientFds[i]);

                        // find maxfd
                        swap(clientFds[clientNum - 1], clientFds[i]);
                        if (clientFds[i] + 1 == maxFd) {
                            maxFd --;
                        }
                        clientNum --;
                    }
                }
            }
        }

    }
}