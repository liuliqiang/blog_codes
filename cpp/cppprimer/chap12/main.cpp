#include <iostream>
#include <memory>

using namespace std;

int main()
{
    shared_ptr<int> ptr(new int(1024));

    cout << *ptr << endl;

    delete ptr.get();
    cout << *ptr << endl;
    return 0;
}