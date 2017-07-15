#include <iostream>
#include <ctime>
#include <csignal>
using namespace std;

#include <zconf.h>
#include <event2/event.h>

void sigCallback(evutil_socket_t fd, short event, void *arg) {
    cout << "In signal callback" << endl;
    event_base *base = (event_base*) arg;
    event_base_loopbreak(base);
}

int main()
{
    event_base *base = event_base_new();

    event *ev = evsignal_new(base, (evutil_socket_t)SIGALRM, sigCallback, base);
    evsignal_add(ev, NULL);

    alarm(5);
    event_base_dispatch(base);

    cout << "exit now" << endl;
    return 0;
}