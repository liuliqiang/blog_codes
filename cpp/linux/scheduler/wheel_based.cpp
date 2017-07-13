//
// Created by yetship on 2017/7/14.
//
#include <iostream>
#include <array>
#include <forward_list>
using namespace std;

#include <time.h>
#include <sys/time.h>

#define WHEEL_SIZE 8
#define TIMER_NUM 5
#define SECONDS 1000000000

typedef int timer_id;
typedef int timer_expiry(timer_id id, void *user_data, int len);

int alarmCnt = 0;
int currTimerId = 0;
timer_id id[TIMER_NUM];

struct timer {
    timer_id id;			/**< timer id		*/

    int interval;			/**< timer interval(second)*/
    int elapse; 			/**< 0 -> interval 	*/

    timer_expiry *cb;		/**< call if expiry 	*/
    void *user_data;		/**< callback arg	*/
    int len;			/**< user_data length	*/

    timer(int interval, timer_expiry *timerCb, void* userData, int dataLen):
            interval(interval), elapse(0), cb(timerCb), id(currTimerId++) {
        if(userData != NULL || dataLen != 0) {
            this->user_data = malloc(dataLen);
            memcpy(this->user_data, userData, dataLen);
            this->len = dataLen;
        } else {
            this->user_data = NULL;
        }
    }
};

int currSlot = 0;
array<forward_list<timer>, WHEEL_SIZE> timeWheel;


time_t addTimer(int interval, timer_expiry *cb, void *user_data, int len) {
    int targetSlot = (interval + WHEEL_SIZE - currSlot) % WHEEL_SIZE;

    timer t = timer(interval, cb, user_data, len);
    timeWheel[targetSlot].push_front(t);

    return t.id;
}

static char *fmt_time(char *tstr)
{
    time_t t;

    t = time(NULL);
    strcpy(tstr, ctime(&t));
    tstr[strlen(tstr)-1] = '\0';

    return tstr;
}

int timer_cb(timer_id id, void *arg, int len)
{
    char tstr[200];
    alarmCnt ++;
    printf("hello [%s]/id %d: timer '%s' cb is here.\n", fmt_time(tstr), id, (char*)arg);
    return 0;
}



/* Tick Bookkeeping */
static void sig_func(int signo)
{
    printf("sig_func with no: %d\n", signo);
    auto curr = timeWheel[currSlot].begin();
    while (curr != timeWheel[currSlot].end()) {
        curr->elapse ++;
        if (curr->elapse >= curr->interval) {
            curr->elapse = 0;
            curr->cb(curr->id, curr->user_data, curr->len);
            timeWheel[currSlot].erase_after(curr++);
        }
    }

    currSlot ++;
}


int main() {
    void (*old_sigfunc)(int);
    if ((old_sigfunc = signal(SIGALRM, sig_func)) == SIG_ERR) {
        printf("signal register error");
        return -1;
    }

    /* Setting our interval timer for driver our mutil-timer and store old timer value */
    struct itimerval value, ovalue;
    value.it_value.tv_sec = 1;
    value.it_value.tv_usec = 0;
    value.it_interval.tv_sec = 1;
    value.it_interval.tv_usec = 0;
    int ret = setitimer(ITIMER_REAL, &value, &ovalue);
    if (ret != 0) {
        perror("set timer error");
        return -1;
    }


    for (int i = 0; i < WHEEL_SIZE; i++) {
        timeWheel[i] = forward_list<timer>();
    }

    printf("register all\n");
    id[0] = addTimer(2, timer_cb, (void*)"a", sizeof("a"));
    id[1] = addTimer(3, timer_cb, (void*)"b", sizeof("b"));
    id[2] = addTimer(5, timer_cb, (void*)"c", sizeof("c"));
    id[3] = addTimer(7, timer_cb, (void*)"d", sizeof("d"));
    id[4] = addTimer(9, timer_cb, (void*)"e", sizeof("e"));

    // set timer
    struct timespec time;
    time.tv_sec = 0;
    time.tv_nsec= 0.1 * SECONDS;
    printf("timer waiting\n");
    while (alarmCnt < 45) {
        nanosleep(&time, NULL);
    }
}
