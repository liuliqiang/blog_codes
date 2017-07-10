#include <iostream>
#include <deque>
using namespace std;

/**
 * 编写程序，从标准输入读取 string 序列，存入一个 deque 中。
 * 编写一个循环，用迭代器打印 deque 中的元素
 */
int main()
{
    deque<string> inputs;

    for (string input; cin >> input; ) {
        inputs.push_back(input);
    }

    for (auto b = inputs.cbegin(); b != inputs.cend(); b ++) {
        cout << *b << endl;
    }
}