#include <iostream>
#include <vector>
#include <list>
using namespace std;

/**
 * 编写程序，将一个 list 中的 char* 指针元素赋值给一个 vector 中的 string
 */
int main()
{
    list<const char*> ss = {"a", "ab", "abc"};
    vector<string> vec;
    vec.assign(ss.cbegin(), ss.cend());
    for (string s: vec) {
        cout << s << endl;
    }

    vector<string> vec2{ss.cbegin(), ss.cend()};
    for (string s: vec2) {
        cout << s << endl;
    }
}