#include <string>
#include <cstdio>
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
    servAddr.sin_family = PF_INET;
    inet_aton(HOST, &servAddr.sin_addr);
    servAddr.sin_port = htons(PORT);

    int cliFd = socket(PF_INET, SOCK_STREAM, 0);
    int connStatus = connect(cliFd, (const sockaddr*)&servAddr, sizeof(servAddr));
    if (connStatus == -1) {
        perror("connect to server error");
        return -1;
    }

    char sendContent[] = "世界和平!";
    write(cliFd, sendContent, strlen(sendContent) + 1);
    shutdown(cliFd, SHUT_WR);

    int readLen;
    char buff[BUFFER_SIZE];
    for (int readLen = 1; (readLen = recv(cliFd, buff, BUFFER_SIZE-1, 0)) != -1;) {
        if (readLen == 0) {
            break;
        }
        cout << "received from server len: " << readLen << endl;
        cout << "received from server: " << buff << endl;
    }
    cout << "recv null form server" << endl;
    close(cliFd);
}