cmake_minimum_required(VERSION 3.6)
project(concurrency)

set(CMAKE_CXX_STANDARD 11)

include_directories("/Users/luke/github/homebrew/Cellar/libevent/2.1.8/include")
link_directories("/Users/luke/github/homebrew/Cellar/libevent/2.1.8/lib")

set(SOURCE_FILES libevent_server.cpp)
add_executable(concurrency ${SOURCE_FILES})
target_link_libraries(concurrency event)