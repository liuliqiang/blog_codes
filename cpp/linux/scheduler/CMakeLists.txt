cmake_minimum_required(VERSION 3.6)
project(scheduler)

set(CMAKE_CXX_STANDARD 11)

set(SOURCE_FILES heap_based.cpp)
add_executable(scheduler ${SOURCE_FILES})