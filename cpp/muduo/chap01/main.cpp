#include <iostream>

#include <boost/core/noncopyable.hpp>
#include <muduo/base/Mutex.h>


class Counter: boost::noncopyable {
public:
    Counter(): value_(0) {}
    int64_t value() const;
    int64_t getAndIncrease();

private:
    int64_t value_;
    mutable MutexLock mutex_;
};

int main()
{
    std::cout << "Hello, World!" << std::endl;
    return 0;
}