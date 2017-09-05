#include "Python.h"

#include <iostream>
using namespace std;


int main()
{
    // !!! Important
    Py_Initialize();

    PyObject *t = PyTuple_New(3);
    PyTuple_SetItem(t, 0, PyLong_FromLong(1L));
    PyTuple_SetItem(t, 1, PyLong_FromLong(2L));
    PyTuple_SetItem(t, 2, PyUnicode_FromString("three"));

    cout << "size of PyTupleObject: " << sizeof(*t) << endl;

    PyObject *num1 = PyLong_FromLong(3L);
    PyObject_Print(num1, stdout, Py_PRINT_RAW);
    cout << endl;

    PyObject_Print(t, stdout, Py_PRINT_RAW);
    cout << endl;

    // !!! Important too
    Py_Finalize();
    return 0;
}
