//
// Created by yetship on 2017/7/13.
//
#include <cstdio>
#include <iostream>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <cstring>

#include <event2/event.h>
#include <zconf.h>
using namespace std;

#define HOST "127.0.0.1"
#define PORT 3002
#define QUEUE_SIZE 5
#define BUFFER_SIZE 1024
#define MAX_CONNECTION 1024
#define DEBUG true


#ifdef DEBUG
    #define pout cout
#else
    #define pout 0 && cout
#endif

struct FdStat {
    char buffer[BUFFER_SIZE];
    size_t bufferUsed;

    size_t writenNum;
    size_t writeUpto;

    event *read_event;
    event *write_event;

    bool closing;
};

void do_read(evutil_socket_t , short , void *);
void do_write(evutil_socket_t , short , void *);


FdStat* allocFdState(event_base *base, evutil_socket_t fd) {
    FdStat *state = new FdStat();
    if (!state) {
        return NULL;
    }

    state->read_event = event_new(base, fd, EV_READ|EV_PERSIST, do_read, state);
    if (!state->read_event) {
        delete state;
        return NULL;
    }

    state->write_event = event_new(base, fd, EV_WRITE|EV_PERSIST, do_write, state);
    if (!state->write_event) {
        event_free(state->read_event);
        delete state;
        return NULL;
    }

    state->bufferUsed = state->writenNum = state->writeUpto = 0;
    state->closing = false;

    return state;
}


void freeFdState(FdStat *state) {
    event_free(state->read_event);
    event_free(state->write_event);
    free(state);
}


void do_read(evutil_socket_t readFd, short event, void *arg) {
    FdStat *state = (FdStat*)arg;
    char buffer[BUFFER_SIZE];

    int readNum;
    for (readNum = 0; true; ) {
        pout << "[.] read from socket: " << readFd << endl;
        /**
         * 因为 readFd 是 Nonblock 的，所以不会阻塞
         * Upon successful completion, recv() shall return the length of the message in bytes.
         * If no messages are available to be received and the peer has performed an orderly shutdown, recv() shall return 0.
         * Otherwise, -1 shall be returned and errno set to indicate the error.
         *
         * EAGAIN or EWOULDBLOCK
         *    The socket's file descriptor is marked O_NONBLOCK and no data is waiting to be received;
         *    or MSG_OOB is set and no out-of-band data is available
         *    and either the socket's file descriptor is marked O_NONBLOCK
         *    or the socket does not support blocking to await out-of-band data.
         */

        readNum = recv(readFd, buffer, BUFFER_SIZE - 1, 0);
        if (readNum > 0) {
            if (state->writeUpto + readNum >= BUFFER_SIZE) {
                perror("[x] buffer not enough");
                break;
            } else {
                pout << "[.] received from " << readFd << " " << buffer << endl;
                memcpy(state->buffer + state->writeUpto, buffer, readNum);
                event_add(state->write_event, NULL);
                state->writeUpto += readNum;
            }
        } else {
            break;
        }
    }

    if (readNum == 0) {
        pout << "[.] read num is 0, free fd";
        if (state->writeUpto == state->writenNum) {
            // no need to write
            close(readFd);
            freeFdState(state);
        } else {
            state->closing = true;
            shutdown(readFd, SHUT_RD);
        }
    } else if (readNum < 0) {
        pout << "[.] read num less than 0" << endl;
        if (errno == EAGAIN) {
            return ;
        }
        perror("[x] recv");
        freeFdState(state);
    }
}

void do_write(evutil_socket_t writeFd, short event, void *arg) {
    pout << "[.] do_write for socket: " << writeFd << endl;
    FdStat *state = (FdStat*) arg;

    while (state->writenNum < state->writeUpto) {
        ssize_t result = send(writeFd, state->buffer + state->writenNum,
                              state->writeUpto - state->writenNum, 0);
        if (result == 0) {
            perror("[.] socket write 0 to client");
            break;
        }
        state->writenNum += result;
    }

    if (state->writenNum == state->writeUpto) {
        state->writenNum = state->writeUpto = 0;
    }
    if (state->closing) {
        close(writeFd);
        freeFdState(state);
    }

    event_del(state->write_event);
}

void do_accept(evutil_socket_t listener, short event, void *arg) {
    sockaddr_in cliAddr;
    socklen_t cliAddrLen = sizeof(cliAddr);
    int cliFd = accept(listener, (sockaddr*)&cliAddr, &cliAddrLen);
    if (cliFd < 0) {
        perror("accept client error");
        return ;
    } else if (cliFd == FD_SETSIZE) {
        close(cliFd);
    } else {
        event_base *base = (event_base*) arg;
        evutil_make_socket_nonblocking(cliFd);
        FdStat *cliState = allocFdState(base, cliFd);
        event_add(cliState->read_event, NULL);
    }
}


int main(int argc, char* args[]) {
    setvbuf(stdout, NULL, _IONBF, 0);

    sockaddr_in servAddr;
    servAddr.sin_family = AF_INET;
    inet_pton(AF_INET, HOST, (char*)&servAddr.sin_addr);
    servAddr.sin_port = htons(PORT);

    evutil_socket_t servFd = socket(PF_INET, SOCK_STREAM, 0);
    if (servFd < 0) {
        perror("[x] create server socket error");
        return -1;
    }

    setsockopt(servFd, SOL_SOCKET, SO_REUSEADDR, NULL, 0);
    int opRst = bind(servFd, (const sockaddr*)&servAddr, (socklen_t )sizeof(servAddr));
    if (opRst != 0) {
        perror("[x] bind server to host and port error");
        return -1;
    }

    evutil_make_socket_nonblocking(servFd);

    opRst = listen(servFd, QUEUE_SIZE);
    if (opRst != 0) {
        perror("[x] listen server socket error");
        return -1;
    }

    // libevent join
    event_base *base = event_base_new();
    event *listener_event = event_new(base, servFd, EV_READ|EV_PERSIST, do_accept, (void*)base);
    event_add(listener_event, NULL);
    event_base_dispatch(base);
}