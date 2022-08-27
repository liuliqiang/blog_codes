#include <iostream>
using namespace std;

int main()
{
    cout << "Hello, World!" << endl;

    cout << "cin.eof: " << cin.eof() << endl;
    cout << "cin.fail: " << cin.fail() << endl;
    cout << "cin.bad: " << cin.bad() << endl;
    cout << "cin.good: " << cin.good() << endl;
    // cout << "cin.clear: " << cin.clear() << endl;
    cout << "cin.rdstate: " << cin.rdstate() << endl;

    return 0;
}