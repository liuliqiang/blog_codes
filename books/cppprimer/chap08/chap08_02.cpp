//
// Created by yetship on 2017/7/9.
//
#include <iostream>
#include <fstream>

using namespace std;

int main() {
    /**
     * 等价于：
     *  ifstream f;
     *  f.open(filename);
     */
    ifstream f("/tmp/wifi-07-08-2017__01:28:50.log");
    cout << f.is_open() << endl;
    f.close();

    ofstream out;
    out.open("/tmp/fstream-test.log");
    if (out) {
        out << "just test for ofstream" << endl;
        out.close();
    }
}