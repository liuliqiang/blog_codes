.PHONY: run bench

run1:
	@echo "Running the program..."
	@g++ -o a.out atomic_try.cpp -pthread
	@./a.out

bench:
	@echo "Running the benchmark..."
	@g++ -o a.out lockfree_queue.cpp queue_bench.cpp -pthread
	@./a.out 8 8