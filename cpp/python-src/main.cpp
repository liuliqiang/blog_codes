#include <iostream>

#include "Python.h"

int main()
{
    PyObject *t = PyTuple_New(3);
    PyTuple_SetItem(t, 0, PyLong_FromLong(1L));
    PyTuple_SetItem(t, 1, PyLong_FromLong(2L));
    PyTuple_SetItem(t, 2, PyUnicode_FromString("three"));

    PyObject_Print(t, stdout, 0);
    return 0;
}