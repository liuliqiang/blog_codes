#include <iostream>
#include <vector>
using namespace std;

/**
 * 编写程序，将一个 list 中的 char* 指针元素赋值给一个 vector 中的 string
 */
int main()
{
    vector<int> vec01 = {1, 2, 3, 4};
    vector<int> vec02 = {1, 2, 4};

    cout << (vec01 == vec02) << endl;
}