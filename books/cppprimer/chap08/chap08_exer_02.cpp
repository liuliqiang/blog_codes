//
// Created by yetship on 2017/7/9.
//
#include <iostream>
using namespace std;

/**
 * 测试函数，调用参数为 cin
 */
istream& readFromStream(istream&);

int main() {
    readFromStream(cin);
}

istream& readFromStream(istream& is) {
    for (char d; is >> d; ) {
        cout << d;
    }
    is.clear();
    return is;
}