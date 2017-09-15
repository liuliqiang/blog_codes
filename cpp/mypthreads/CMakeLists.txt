cmake_minimum_required(VERSION 3.6)
project(mypthreads)

set(CMAKE_CXX_STANDARD 11)

# set(SOURCE_FILES pthread-demo-01.cpp)
set(SOURCE_FILES pthread-demo-08.cpp)
add_executable(mypthreads ${SOURCE_FILES})