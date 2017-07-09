//
// Created by yetship on 2017/7/9.
//
#include <iostream>
#include <fstream>
#include <vector>
using namespace std;

/**
 * 重写 8.4 的程序，将每个单词作为一个独立的元素进行存储
 * 注解：可以通过 wc 来核对
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
    for (string word; f >> word; ) {
        lineVector.push_back(word);
    }

    cout << "total lines: " << lineVector.size() << endl;
    cout << "first lines: " << lineVector[0] << endl;
    cout << "last lines: " << lineVector[lineVector.size() - 1] << endl;
}