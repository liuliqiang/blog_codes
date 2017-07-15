#include <string>
#include <cstdio>
#include <cstring>
#include <iostream>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <zconf.h>

using namespace std;

#define HOST "192.168.199.138"
#define PORT 3002
#define QUEUE_SIZE 5
#define BUFFER_SIZE 1024
#define SEND_LEN 20

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

    char sendContent[SEND_LEN] = "世界和平!";
    send(cliFd, sendContent, strlen(sendContent) + 1, 0);

    int readLen;
    char buff[BUFFER_SIZE];
    for (int readLen = 1; true;) {
        cout << "[.] read from server" << endl;
        readLen = recv(cliFd, buff, BUFFER_SIZE-1, 0);
        if (readLen <= 0) {
            cout << "server close" << endl;
            break;
        }
        cout << "received from server len: " << readLen << endl;
        cout << "received from server: " << buff << endl;

        if (readLen < SEND_LEN) {
            cout << "cat string: " << strcat(sendContent, "!") << endl;
            cout << "str len: " << strlen(sendContent) << endl;
            cout << "send: " << send(cliFd, sendContent, strlen(sendContent) + 1, 0) << endl;
        } else {
            shutdown(cliFd, SHUT_WR);
        }

        sleep(1);
    }
    cout << "recv null form server" << endl;
    close(cliFd);
}