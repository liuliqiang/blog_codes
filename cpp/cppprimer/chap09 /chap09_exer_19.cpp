#include <iostream>
#include <list>
using namespace std;

/**
 * 重写上题的程序，用 list 代替 deque，列出程序要作出哪些改变
 */
int main()
{
    // 这里做了一个改变
    list<string> inputs;

    for (string input; cin >> input; ) {
        inputs.push_back(input);
    }

    for (auto b = inputs.cbegin(); b != inputs.cend(); b ++) {
        cout << *b << endl;
    }
}