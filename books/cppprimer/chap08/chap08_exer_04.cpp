//
// Created by yetship on 2017/7/9.
//
#include <iostream>
#include <fstream>
#include <vector>
using namespace std;

/**
 * 编写函数，以读模式打开一个文件，将其内容读入到一个 string 的 vector 中
 * 将每一行作为一个独立的元素存于 vector 中
 */

#define MAX_LEN 1024

int main() {
    ifstream f("/tmp/fstream.txt");
    vector<string> lineVector;

    /**
     * getline 的函数原型为：
     *      istream& getline (char* s, streamsize n );
     *      istream& getline (char* s, streamsize n, char delim );
     */
    for (char data[MAX_LEN + 1]; f.getline(data, MAX_LEN); ) {
        lineVector.push_back(data);
    }

    cout << "total lines: " << lineVector.size() << endl;
    cout << "first lines: " << lineVector[0] << endl;
    cout << "last lines: " << lineVector[lineVector.size() - 1] << endl;
}