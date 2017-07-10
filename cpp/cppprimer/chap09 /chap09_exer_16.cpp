#include <iostream>
#include <vector>
#include <list>
using namespace std;

/**
 * 编写程序，将一个 list 中的 char* 指针元素赋值给一个 vector 中的 string
 */
int main()
{
    list<int> vec01 = {1, 2, 3, 4};
    vector<int> vec02 = {1, 2, 4};

    cout << (vector<int>(vec01.cbegin(), vec01.cend()) == vec02) << endl;
}