#include <iostream>
#include <vector>
using namespace std;


vector<int>::iterator& findElemInIter(vector<int>::iterator &begin,
                                     vector<int>::iterator &end,
                                     int val) {
    while (begin != end) {
        if (*begin != val) {
            begin ++;
        } else {
            return begin;
        }
    }
    return end;
}

void foundElem(vector<int> intVec, int elem) {
    auto begin = intVec.begin();
    auto end = intVec.end();
    auto rst = findElemInIter(begin, end, elem);
    if (rst == end) {
        cout << "[x] " << " not found" << endl;
    } else {
        cout << "[.] " << "fount it" << endl;
    }
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

    foundElem(intVec, 3);
    foundElem(intVec, 7);
    foundElem(intVec, 10);
}