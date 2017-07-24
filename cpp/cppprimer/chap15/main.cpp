#include <iostream>
using namespace std;


class A {
    public:
    void a() {
        cout << "A.a" << endl;
    }

    virtual void b() {
        cout << "A.b" << endl;
    }
};


class B: public A {
    public:
    virtual void a() {
        cout << "B.a" << endl;
    }

    void b() {
        cout << "B.b" << endl;
    }
};
int main()
{
    A a;
    a.a();
    a.b();
    B b;
    a = b;
    a.a();
    a.b();

    A &c = b;
    c.a();
    c.b();
    return 0;
}