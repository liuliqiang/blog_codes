cmake_minimum_required(VERSION 3.6)
project(concurrency)

set(CMAKE_CXX_STANDARD 11)

set(SOURCE_FILES select_server.cpp)
add_executable(concurrency ${SOURCE_FILES})