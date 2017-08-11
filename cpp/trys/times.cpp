#include <iostream>
#include <ctime>
#include <chrono>
using namespace std;

int main()
{
    chrono::milliseconds ms = chrono::duration_cast<chrono::milliseconds >(
        chrono::system_clock::now().time_since_epoch());

    cout << ms.count() << endl;

    time_t nowTime = time(0);
    tm *local=localtime(&nowTime);  //获取当前系统时间

    char buf[80];
    strftime(buf,80,"格式化输出：%Y-%m-%d %H:%M:%S",local);
    cout << buf << endl;


    return 0;
}
