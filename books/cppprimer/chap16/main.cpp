#include <iostream>
using namespace std;


template <typename T>
bool compare(const T &a, const T &b) {
    return a < b;
}

int main()
{
    cout << compare(1, 2) << endl;
    cout << compare(2, 1) << endl;
    cout << compare<int>(2, 1) << endl;
    return 0;
}