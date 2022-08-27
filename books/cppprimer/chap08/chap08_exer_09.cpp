//
// Created by yetship on 2017/7/9.
//
#include <iostream>
#include <sstream>
#include <vector>
using namespace std;

/**
 * 使用你为 8.1.2 节第一个练习所编写的函数打印一个 istringstream 对象的内容
 */

istream& readFromStream(istream& is) {
    for (char d; is.get(d); ) {
        cout << d;
    }
    is.clear();
    return is;
}

int main(int argc, char* args[]) {
    stringstream ss("I am the king of the world");

    readFromStream(ss);
}