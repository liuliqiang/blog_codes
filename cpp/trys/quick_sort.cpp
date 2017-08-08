#include <cstdlib>
#include <ctime>

#include <iostream>
using namespace std;


void quickSort(int *ary, int start, int end) {
    int pivot = ary[start];

    int l = start, r = end;
    while (l <= r) {
        while (l <= r && ary[l] < pivot)
            l ++;
        while (l <= r && ary[r] > pivot)
            r --;
        if (l <= r) {
            swap(ary[r--], ary[l++]);
        }
    }

    if (start < r) {
        quickSort(ary, start, r);
    }
    if (l < end) {
        quickSort(ary, l, end);
    }
}


int main() {
    time_t t = time(0);
    srand(t);
    int ary[11];

    for (int r = 5; r < 11; r ++) {
        for (int i = 0; i < r; i++) {
            ary[i] = rand() % 10;
            cout << ary[i] << " ";
        }
        cout << endl;

        quickSort(ary, 0, r - 1);
        for (int i = 0; i < r; i++) {
            cout << ary[i] << " ";
        }
        cout << endl;
    }
}
