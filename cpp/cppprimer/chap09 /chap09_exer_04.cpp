#include <iostream>
#include <vector>
using namespace std;


bool findElemInIter(vector<int>::iterator begin, vector<int>::iterator end, int val) {
    while (begin != end) {
        if (*begin != val) {
            begin ++;
        } else {
            return true;
        }
    }
    return false;
}
int main()
{
    vector<int> intVec;

    intVec.push_back(1);
    intVec.push_back(2);
    intVec.push_back(3);
    intVec.push_back(4);
    intVec.push_back(5);
    intVec.push_back(6);
    intVec.push_back(7);

    cout << "find 3 in iterator range: " << findElemInIter(intVec.begin(), intVec.end(), 3) << endl;
    cout << "find 7 in iterator range: " << findElemInIter(intVec.begin(), intVec.end(), 7) << endl;
    cout << "find 10 in iterator range: " << findElemInIter(intVec.begin(), intVec.end(), 10) << endl;
}