#include<iostream>

#include <sys/signal.h>
#include <event.h>


void signalCallback(evutil_socket_t fd, short event, void *argc) {
    event_base *base = (event_base*) argc;
    timeval delay = {2, 0};

    std::cout << "Caught an interrupt; exiting in two seconds" << std::endl;
    event_base_loopexit(base, &delay);
}


void timeoutCallback(int fd, short event, void *argc) {
    std::cout << "Timeout" << std::endl;
}


int main() {
    event_base *base = event_init();
    // event_base 支持优先级
    int priority = event_base_priority_init(base, 5);
    if (priority == -1) {
        std::cerr << "Event base 不支持优先级" << std::endl;
    }
    std::cout << "Max Priority: " << EVENT_MAX_PRIORITIES << std::endl << std::endl;


    event *signalEvent = evsignal_new(base, SIGINT, signalCallback, base);
    event_add(signalEvent, NULL);

    timeval time = {1, 0};
    event *timeoutEvent = evtimer_new(base, timeoutCallback, base);
    event_add(timeoutEvent, &time);

    // 获得所有的支持的IO复用方法
    const char* *supportMethods = event_get_supported_methods();
    while (*supportMethods) {
        std::cout << "Support Method: " << *supportMethods << std::endl;
        supportMethods ++;
    }

    // 查看当前使用的 IO复用方法
    const char *usingMethod = event_base_get_method(base);
    std::cout << std::endl;
    std::cout << "Current Method: " << usingMethod << std::endl;
    std::cout << std::endl;

    // 查看当前使用的 IO复用特性
    int feature = event_base_get_features(base);
    if (feature & EV_FEATURE_ET) {
        std::cout << "Support Feature: ET" << std::endl;
    } else if (feature & EV_FEATURE_FDS) {
        std::cout << "Support Feature: FDS" << std::endl;
    } else if (feature & EV_FEATURE_EARLY_CLOSE) {
        std::cout << "Support Feature: EARLY_CLOSE" << std::endl;
    } else if (feature & EV_FEATURE_O1) {
        std::cout << "Support Feature: O1" << std::endl;
    }

    // dump event 列表到文件中
    FILE *fp = fopen("/tmp/event_list.log", "w");
    event_base_dump_events(base, fp);

    event_base_dispatch(base);

    // 查看 event_base 退出状态
    std::cout << std::endl << "Does event base exit: " << event_base_got_exit(base) << std::endl;
    std::cout << "Does event base break: " << event_base_got_break(base) << std::endl;

    // 释放空间
    event_free(timeoutEvent);
    event_free(signalEvent);
    event_base_free(base);
}
