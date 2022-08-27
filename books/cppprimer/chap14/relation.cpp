//
// Created by yetship on 2017/7/17.
//
#include <cstdio>
#include <cstdlib>
#include <cstring>

#include <string>
#include <iostream>
using namespace std;


class BigInteger {
public:
    string digits;
    BigInteger() = default;
    BigInteger(int num) {
        digits = to_string(num);
    }

    BigInteger plus(BigInteger &bi) {
        BigInteger rtnValue;
        size_t myLen = this->digits.length();
        size_t addedNumLen = bi.digits.length();
        const unsigned long len = min(myLen, addedNumLen);
        const unsigned long maxLen = max(myLen, addedNumLen);

        char result[maxLen + 2];
        bzero(result, maxLen + 2);

        size_t idx = maxLen;
        for(; myLen > 0 && addedNumLen > 0; myLen --, addedNumLen --, idx--) {
            result[idx] += (this->digits[myLen-1] + bi.digits[addedNumLen-1] - '0' * 2);
            result[idx-1] += result[idx] / 10;
            result[idx] = result[idx] % 10 + '0';
        }
        for (; myLen > 0; myLen--, idx--) {
            result[idx] += (this->digits[myLen - 1] - '0');
            result[idx - 1] += result[idx] / 10;
            result[idx] = result[idx] % 10 + '0';
        }
        for (; addedNumLen > 0; addedNumLen--, idx--) {
            result[idx] += (bi.digits[addedNumLen--] - '0');
            result[idx - 1] += result[idx] / 10;
            result[idx] = result[idx] % 10 + '0';
        }

        if (result[0]) {
            cout << "result 1 " << result << endl;
            rtnValue.digits = string(result);
        } else {
            cout << "result 0 " << &result[1] << endl;
            rtnValue.digits = string(&result[1]);
        };
        return rtnValue;
    }
};

BigInteger operator+(BigInteger &bi1, BigInteger &bi2) {
    return bi1.plus(bi2);
}

ostream& operator<<(ostream& os, const BigInteger &bi) {
    os << bi.digits;
    return os;
}


int main() {
    BigInteger a(1), b(2);

    BigInteger c = a.plus(b);
    cout << c << endl;

    cout << (a + b) << endl;
}
