//
// Created by yetship on 2017/8/4.
//
#include <cstdio>
#include <cstdlib>
#include <iostream>
using  namespace std;

#include "apue.h"

int main() {
    size_t n;
    char buff[MAXLINE], command[MAXLINE];
    FILE *fp;

    fgets(buff, MAXLINE, stdin);
    n = strlen(buff);
    if (buff[n-1] == '\n') {
        n --;
    }

    snprintf(command, sizeof(command), "cat %s", buff);
    fp = popen(command, "r");

    while (fgets(buff, MAXLINE, fp) != NULL) {
        fputs(buff, stdout);
    }

    pclose(fp);
    exit(0);
}

