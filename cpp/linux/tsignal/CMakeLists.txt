cmake_minimum_required(VERSION 3.6)
project(tsignal)

set(CMAKE_CXX_STANDARD 11)

include_directories("/Users/luke/github/homebrew/Cellar/libevent/2.1.8/include")
link_directories("/Users/luke/github/homebrew/Cellar/libevent/2.1.8/lib")


set(SOURCE_FILES main.cpp)
add_executable(tsignal ${SOURCE_FILES})
target_link_libraries(tsignal event)
