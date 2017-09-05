cmake_minimum_required(VERSION 3.6)
project(python_src)

find_package(PythonLibs REQUIRED)
include_directories(${PYTHON_INCLUDE_DIRS})

set(CMAKE_CXX_STANDARD 11)

set(SOURCE_FILES main.cpp)

add_executable(python_src ${SOURCE_FILES})
target_link_libraries(python_src ${PYTHON_LIBRARIES})
