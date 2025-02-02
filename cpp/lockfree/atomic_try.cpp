#include <atomic>
#include <iostream>
#include <thread>

std::atomic<int> counter(0);

void increment()
{
    for (int i = 0; i < 1000; ++i)
    {
        // counter.fetch_add(1, std::memory_order_relaxed);
        counter++;
    }
}

int main()
{
    std::thread t1(increment);
    std::thread t2(increment);

    t1.join();
    t2.join();

    std::cout << counter << std::endl;

    return 0;
}