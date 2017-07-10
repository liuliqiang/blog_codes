#include <iostream>
#include <list>
#include <deque>
#include <vector>
using namespace std;

/**
 * 编写程序，分别使用 at／下标运算符／front 和 begin 提取一个 vector 中的第一个元素
 * 在一个空 vector 上测试你的程序
 *
 * 结果应该是代码以错误码退出程序
 */
int main()
{
    vector<int> emptyVector;

    cout << emptyVector[0] << endl;
    cout << emptyVector.at(0) << endl;
    cout << emptyVector.front() << endl;
    cout << *emptyVector.begin() << endl;
}