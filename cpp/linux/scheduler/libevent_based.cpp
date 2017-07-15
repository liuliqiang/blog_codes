//
// Created by yetship on 2017/7/14.
//
#include <iostream>
using namespace std;

#include <event2/event.h>


void sayHello(evutil_socket_t fd, short event, void *arg) {
    cout << "Hello in signal callback" << endl;
}

int main() {
    event_base *base = event_base_new();

    event *ev = evtimer_new(base, sayHello, NULL);

}
