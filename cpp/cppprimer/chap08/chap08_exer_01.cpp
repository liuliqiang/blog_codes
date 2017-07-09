//
// Created by yetship on 2017/7/9.
//
#include <iostream>
using namespace std;

/**
 * 编写函数，接受一个 istream& 参数，返回值类型也是 istream&。此函数须从给定流中读取数据，
 * 直至遇到文件结束标识时停止。它将读取的数据打印在标准输出上。
 * 完成这些操作后，在返回流之前，对流进行复位，使其处于有效状态。
 */
istream& readFromStream(istream&);

int main() {
    readFromStream(cin);
}

istream& readFromStream(istream& is) {
    for (char d; is.get(d); ) {
        cout << d;
    }
    is.clear();
    return is;
}