//
// Created by yetship on 2017/7/3.
//
#include <iostream>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <sys/socket.h>
#include <netdb.h>

using namespace std;


union {
    char vals[4] = {'l', '?', '?', 'b'};
    long c;
} test;

int main() {
    cout << (char)test.c << endl;

    cout << __DARWIN_BYTE_ORDER << endl;
    cout << __DARWIN_LITTLE_ENDIAN << endl;
    cout << __DARWIN_BIG_ENDIAN << endl;

    sockaddr_in addr;
    addr.sin_family = AF_INET;
    addr.sin_port = htons(1080);

    cout << "liuliqiang.info host: " << *(gethostbyname("liuliqiang.info")->h_addr_list) << endl;
    cout << "127.0.0.1 name" << gethostbyaddr("127.0.0.1", strlen("127.0.0.1"), AF_INET) << endl;

    struct in_addr s;
    char src[32];
    cout << "inet_pton 127.0.0.1: " << endl;
    cout << "\treturn is: " << inet_pton(AF_INET, "127.0.0.1", &s) << endl;
    cout << "\tresult is: " << s.s_addr << endl;

    cout << "inet_ntop: " << endl;
    cout << "\treturn is: " << inet_ntop(AF_INET, &s, src, INET_ADDRSTRLEN) << endl;
    cout << "\tresult is: " << src << endl;

    int servFd = socket(AF_INET, SOCK_STREAM, 0);

}
