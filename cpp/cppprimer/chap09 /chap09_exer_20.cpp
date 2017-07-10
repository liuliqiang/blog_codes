#include <iostream>
#include <list>
#include <deque>
#include <vector>
using namespace std;

/**
 * 编写程序，从一个 list<int> 拷贝元素到两个 deque 中。
 * 值为偶数的所有元素都拷贝到一个 deque 中，而奇数值都拷贝到另一个 deque 中
 */
int main()
{
    list<int> intList = {1, 2, 3, 4, 5, 6, 7};
    vector<deque<int>> splitDeque(2);

    for (int elem: intList) {
        splitDeque[elem % 2].push_back(elem);
    }

    cout << "deque 1 中的元素有: ";
    for (int elem: splitDeque[0]) {
        cout << elem << " ";
    }
    cout << endl << "deque 2 中的元素有：";
    for (int elem: splitDeque[1]) {
        cout << elem << " ";
    }
    cout << endl;
}