#include <iostream>
using namespace std;


class Personal {
public:
    string firstName;
    string lastName;
    int age;

    Personal() = default;
    Personal(string fn, string ln, int age): firstName(fn), lastName(ln), age(age) {};
};

ostream &operator<< (ostream &os, const Personal &p) {
    os << "Name: " << p.lastName << " " << p.firstName << " Age: " << p.age << endl;
    return os;
}

istream &operator>> (istream &is, Personal &p) {
    is >> p.firstName >> p.lastName >>p.age;
    return is;
}

int main()
{
    Personal p("liqiang", "lau", 26);
    Personal p2;

    cout << p << endl;

    cin >> p2;
    cout << p2 << endl;
    return 0;
}