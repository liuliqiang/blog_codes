#include <iostream>
#include <atomic>
#include <memory>
#include <thread>
#include <vector>
#include <chrono>

#include "lockfree_queue.cpp"

// Benchmarking function
void producer(LockFreeQueue<int>& queue, int count) {
    for (int i = 0; i < count; ++i) {
        queue.enqueue(i);
    }
}

void consumer(LockFreeQueue<int>& queue, std::atomic<int>& totalCount) {
    int value;
    while (totalCount > 0) {
        if (queue.dequeue(value)) {
            totalCount--;
        }
    }
}

int main(int argc, char* argv[]) {
    if (argc < 3) {
        std::cerr << "Usage: " << argv[0] << " <num_producers> <num_consumers>" << std::endl;
        return 1;
    }

    int numProducers = std::stoi(argv[1]);
    int numConsumers = std::stoi(argv[2]);
    int itemsPerProducer = 100000;
    std::atomic<int> totalItems(numProducers * itemsPerProducer);

    LockFreeQueue<int> queue;
    std::vector<std::thread> producers, consumers;

    auto start = std::chrono::high_resolution_clock::now();

    for (int i = 0; i < numProducers; ++i) {
        producers.emplace_back(producer, std::ref(queue), itemsPerProducer);
    }
    for (int i = 0; i < numConsumers; ++i) {
        consumers.emplace_back(consumer, std::ref(queue), std::ref(totalItems));
    }

    for (auto& p : producers) p.join();
    for (auto& c : consumers) c.join();

    auto end = std::chrono::high_resolution_clock::now();
    std::chrono::duration<double> elapsed = end - start;
    
    std::cout << "Benchmark completed in " << elapsed.count() << " seconds" << std::endl;
    std::cout << "Throughput: " << (numProducers * itemsPerProducer) / elapsed.count() << " ops/sec" << std::endl;

    return 0;
}
