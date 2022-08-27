#include <iostream>
#include <ctime>

using namespace std;


bool binary_search(int *ary, int begin, int end, int elem) {
    // 1. 校验参数非常重要
    if (begin > end) {
        return -1;
    }

    int mid = begin;
    while (begin <= end) {
        // 2. 如果使用 mid = (begin + end) / 2; 有溢出风险
        mid = begin + (end - begin) / 2;

        if (ary[mid] == elem) {
            return true;
        } else if (ary[mid] > elem) {
            end = mid - 1;
        } else {
            begin = mid + 1;
        }
    }

    return false;
}


int newCompare(const void*a, const void *b, int flag) {
    // 当 flag = 1 时，equal = 0 应该 return 1
    // 当 flag = 0 时，equal = 0 应该 return -1
    int result = (*(int*)a - *(int*)b);
    return result? result: (flag ? flag: flag - 1);
}


int compare(const void *a, const void *b) {
    return (*(int*)a - *(int*)b);
}


/**
 * 给定一个升序排列的自然数数组，数组中包含重复数字，例如：[1,2,2,3,4,4,4,5,6,7,7]。
 * 问题：给定任意自然数，对数组进行二分查找，返回数组正确的位置，给出函数实现。
 * 注：连续相同的数字，返回第一个匹配位置还是最后一个匹配位置，由函数传入参数决定。
 * flag: 1 返回第一个等值项
 * flag: 0 返回最后一个等值项
 *
 * Yetship Comment:
 *    复杂度稳定在: O(log(n))
 */
bool binarySearchWithIdx(int *ary, int size, int key, int flag, int &position) {
    if (size < 0) {
        return false;
    }
    int low = 0,
        high = size - 1,
        midPos = 0;

    while (low <= high) {
        midPos = low + (high - low) / 2;

        if (ary[midPos] == key) {
            position = midPos;
            return true;
        } else if (ary[midPos] < key) {
            low = midPos + 1;
        } else {
            high = midPos - 1;
        }
    }

    return false;
}


int main() {
    time_t t = time(0);
    srand(t);

    int ary[11];
    for (int i = 0; i < 11; i++) {
        ary[i] = rand() % 10;
    }

    qsort(ary, 11, sizeof(int), compare);
    for (int i = 0; i < 11; i++) {
        cout << ary[i] << " ";
    }
    cout << endl;
    cout << binary_search(ary, 0, 11, 8) << endl;
    cout << binary_search(ary, 0, 7, 8) << endl;
    cout << binary_search(ary, 3, 11, 8) << endl;
    cout << binary_search(ary, 7, 11, 8) << endl;
    cout << binary_search(ary, 0, 3, 8) << endl;

    cout << "search in {1}" << endl;
    int oneAry[] = {1};
    cout << binary_search(oneAry, 0, 0, 1) << endl;
    cout << binary_search(oneAry, 0, 0, 0) << endl;
    cout << binary_search(oneAry, 0, 0, 2) << endl;

    cout << "search in {1, 3}" << endl;
    int twoAry[] = {1, 3};
    cout << binary_search(twoAry, 0, 1, 0) << endl;
    cout << binary_search(twoAry, 0, 1, 1) << endl;
    cout << binary_search(twoAry, 0, 1, 2) << endl;
    cout << binary_search(twoAry, 0, 1, 3) << endl;
    cout << binary_search(twoAry, 0, 1, 4) << endl;
    cout << binary_search(twoAry, 0, 1, 5) << endl;

    cout << " test compare " << endl;
    int a = 1, b = 3, c = 3;
    cout << newCompare(&a, &b, 0) << endl;
    cout << newCompare(&a, &b, 1) << endl;
    cout << newCompare(&b, &a, 0) << endl;
    cout << newCompare(&b, &a, 1) << endl;
    cout << newCompare(&c, &b, 0) << endl;
    cout << newCompare(&c, &b, 1) << endl;
}
