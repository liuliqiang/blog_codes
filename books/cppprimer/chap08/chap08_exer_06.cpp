//
// Created by yetship on 2017/7/9.
//
#include <iostream>
#include <fstream>
#include <vector>
using namespace std;

#include "Sales_item.h"

/**
 * 重写 7.1.1 节的书店程序，从一个文件中读取交易记录。
 * 将文件名作为一个参数传递给 main。
 */

int main(int argc, char* args[]) {
    if (argc != 2) {
        cerr << "Usage: " << args[0] << " filename" << endl;
        return -1;
    }

    ifstream f(args[1]);
    Sales_item total;

    if (f >> total) {
        Sales_item trans;
        while (f >> trans) {
            if (total.isbn() == trans.isbn())
                total += trans;
            else {
                cout << total << endl;
                total = trans;
            }
        }

        cout << total << endl;
    } else {
        cout << "No data?!" << endl;
        return -1;
    }

    return 0;

}