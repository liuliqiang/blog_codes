#include <iostream>
#include <vector>
using namespace std;

/**
 * 对 6 种创建和初始化 vector 对象的方法，每一种都给出一个实例。
 * 解释每个 vector 包含什么值
 */
int main()
{
    vector<int> vec01;
    cout << "1. vector<int> vec 里面没有值，size 等于: " << vec01.size() << endl;

    vector<int> vec02 = {1, 2, 3, 4};
    cout << "2. vector<int> vec = {1, 2, 3, 4} 里面有 4 个值，分别是 1， 2， 3， 4" << endl;
    cout << "\t size: " << vec02.size() << " 第1个值: " << vec02[0] << " 最后一个值: " << vec02[3] << endl;

    vector<int> vec03 = vec02;
    cout << "3. vector<int> vec = aotVec 里面的值和上面一样" << endl;
    cout << "\t size: " << vec03.size() << " 第1个值: " << vec03[0] << " 最后一个值: " << vec03[3] << endl;

    vector<int> vec04{vec03.cbegin(), vec03.cend()};
    cout << "4. vector<int> vec{iterator, iterator} 里面的值也和 2 一样" << endl;
    cout << "\t size: " << vec04.size() << " 第1个值: " << vec04[0] << " 最后一个值: " << vec04[3] << endl;

    vector<int> vec05(2);
    cout << "5. vector<int> vec(2) 里面有元素: " << vec05.size() << endl;
    cout << "\t 第一个元素为：" << vec05[0] << " 第二个元素为: " << vec05[1] << endl;

    vector<int> vec06(2, 1);
    cout << "6. vector<int> vec(2, 1) 里面有元素: " << vec06.size() << endl;
    cout << "\t 第一个元素为：" << vec06[0] << " 第二个元素为: " << vec06[1] << endl;
}