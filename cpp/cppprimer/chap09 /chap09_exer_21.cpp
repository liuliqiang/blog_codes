#include <iostream>
#include <list>
#include <deque>
#include <vector>
using namespace std;

/**
 * 编写程序，从一个 list<int> 拷贝元素到两个 deque 中。
 * 值为偶数的所有元素都拷贝到一个 deque 中，而奇数值都拷贝到另一个 deque 中
 */
int main()
{
    string word;
    vector<string> vt;
    auto iter = vt.begin();

    while (cin >> word) {
        if (word == "end")
            break;
        iter = vt.insert(iter, word);
    }

    for (string elem: vt) {
        cout << elem << endl;
    }
}