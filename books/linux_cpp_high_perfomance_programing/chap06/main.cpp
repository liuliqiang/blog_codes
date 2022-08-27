#include <iostream>
#include <vector>
using namespace std;

class Point{
public:
    Point() {
        cout << "Construction" << endl;
    }

    Point(const Point& p) {
        cout << "copy construction" << endl;
    }

    ~Point() {
        cout << "Decostruction" << endl;
    }
};
int main()
{
    int v1 = 10, v2;
    cout << "v1 is: " << v1 << endl;
    cout << "v2 is: " << v2 << endl;

    v2 = 11;
    cout << "v2 is: " << v2 << endl;

    std::cout << "Hello, World!" << std::endl;

    vector<Point> vecPoint(0);
    Point a;
    Point b;
    Point c;

    cout << "push a" << endl;
    vecPoint.push_back(a);

    cout << "push b" << endl;
    vecPoint.push_back(b);

    cout << "push c" << endl;
    vecPoint.push_back(c);

    cout << "vector cout is: " << vecPoint.size() << endl;

    return 0;
}