#include <iostream>
#include <atomic>
#include <memory>

template<typename T>
class LockFreeQueue {
private:
    struct Node {
        T data;
        std::atomic<Node*> next;
        Node(const T& value) : data(value), next(nullptr) {}
    };

    std::atomic<Node*> head;
    std::atomic<Node*> tail;

public:
    LockFreeQueue() {
        Node* sentinel = new Node(T()); // 哨兵节点
        head.store(sentinel);
        tail.store(sentinel);
    }

    ~LockFreeQueue() {
        Node* node;
        while ((node = head.load()) != nullptr) {
            head.store(node->next.load());
            delete node;
        }
    }

    void enqueue(const T& value) {
        Node* newNode = new Node(value);  // ✅ 直接用裸指针，避免 unique_ptr 提前销毁
        Node* oldTail;
        Node* next;
        while (true) {
            oldTail = tail.load();
            next = oldTail->next.load();
            if (oldTail != tail.load()) {
                continue;
            }
            if (next != nullptr) {
                // ✅ 尝试推进 tail
                tail.compare_exchange_weak(oldTail, next);
                continue;
            }
            if (oldTail->next.compare_exchange_weak(next, newNode)) {
                break;
            }
        }
        tail.compare_exchange_weak(oldTail, newNode);  // ✅ 让 tail 指向新节点
    }

    bool dequeue(T& result) {
        Node* oldHead;
        Node* next;
        while (true) {
            oldHead = head.load();
            Node* oldTail = tail.load();
            next = oldHead->next.load();
            if (oldHead != head.load()) {
                continue;
            }
            if (oldHead == oldTail) {
                if (next == nullptr) {
                    return false;  // ✅ 空队列，返回 false
                }
                // ✅ tail 落后了，推进 tail
                tail.compare_exchange_weak(oldTail, next);
                continue;
            }
            if (next == nullptr) {  // ✅ 避免访问 nullptr
                return false;
            }
            if (head.compare_exchange_weak(oldHead, next)) {
                break;
            }
        }
        result = std::move(next->data);  // ✅ next 一定不为 nullptr
        delete oldHead;
        return true;
    }
};
