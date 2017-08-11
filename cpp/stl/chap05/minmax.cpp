#include <iostream>
#include <algorithm>
using namespace std;


bool intPtrLess(int *a, int *b) {
    return *a < *b;
}


int main() {
    int x = 17;
    int y = 42;
    int z = 33;

    int *px = &x;
    int *py = &y;
    int *pz = &z;

    int *pmax = max(px, py, intPtrLess);
    cout << "max(px, py, intPtrLess) = " << *pmax << endl;

    pair<int*, int*> extremes = minmax({px, py, pz}, intPtrLess);

    cout << "minmax() = " << *(extremes.first) << " " << *(extremes.second) << endl;
}
