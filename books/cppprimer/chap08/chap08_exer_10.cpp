//
// Created by yetship on 2017/7/9.
//
#include <iostream>
#include <fstream>
#include <sstream>
#include <vector>
using namespace std;

/**
 * 编写程序，将来自一个文件中的行保存在一个 vector<string> 中，
 * 然后使用一个 istringstream 从 vector 读取数据元素，每次读取一个单词。
 */

int main(int argc, char* args[]) {
    vector<string> lines;
    fstream f("/tmp/fstream.txt", fstream::in);

    for (string line; getline(f, line); ){
        lines.push_back(line);
    }

    for (string line: lines) {
        stringstream ss(line);
        for (string word; ss >> word; ) {
            cout << word << endl;
        }
    }
}