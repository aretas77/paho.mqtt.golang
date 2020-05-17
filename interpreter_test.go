package mqtt

import (
	"os"
	"testing"

	"github.com/DataDog/go-python3"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Setenv("PYTHONPATH", "./")
	os.Exit(m.Run())
}

func TestPythonHermesInitialization(t *testing.T) {
	randomString := "random"

	python3.Py_Initialize()
	//defer python3.Py_Finalize()

	// get the module named interpreter
	obj := python3.PyImport_ImportModule("interpreter")
	assert.NotNil(t, obj)
	defer obj.DecRef()

	// encode a string as PyUnicode type obj
	args := python3.PyUnicode_FromString(randomString)
	assert.True(t, python3.PyUnicode_Check(args))
	defer args.DecRef()

	callable := python3.PyUnicode_FromString("test")
	assert.True(t, python3.PyUnicode_Check(callable))
	defer callable.DecRef()

	// call method `test` with arguments and capture result.
	out := obj.CallMethodObjArgs(callable, args)
	assert.True(t, python3.PyUnicode_Check(out))
	assert.Equal(t, randomString, python3.PyUnicode_AsUTF8(out))
}

func TestPythonHermesCallMethod(t *testing.T) {
	python3.Py_Initialize()
	//defer python3.Py_Finalize()

	s := python3.PyUnicode_FromString("hello world")
	assert.True(t, python3.PyUnicode_Check(s))
	defer s.DecRef()

	sep := python3.PyUnicode_FromString(" ")
	assert.True(t, python3.PyUnicode_Check(sep))
	defer sep.DecRef()

	split := python3.PyUnicode_FromString("split")
	assert.True(t, python3.PyUnicode_Check(split))
	defer split.DecRef()

	words := s.CallMethodObjArgs(split, sep)
	assert.True(t, python3.PyList_Check(words))
	defer words.DecRef()
	assert.Equal(t, 2, python3.PyList_Size(words))

	hello := python3.PyList_GetItem(words, 0)
	assert.True(t, python3.PyUnicode_Check(hello))
	world := python3.PyList_GetItem(words, 1)
	assert.True(t, python3.PyUnicode_Check(world))

	assert.Equal(t, "hello", python3.PyUnicode_AsUTF8(hello))
	assert.Equal(t, "world", python3.PyUnicode_AsUTF8(world))

	words.DecRef()
}

func TestPythonHermesModelInference(t *testing.T) {
	python3.Py_Initialize()
	//defer python3.Py_Finalize()

	obj := python3.PyImport_ImportModule("interpreter")
	assert.NotNil(t, obj)
	defer obj.DecRef()

	callable := python3.PyUnicode_FromString("test_inference")
	assert.True(t, python3.PyUnicode_Check(callable))
	defer callable.DecRef()

	binaryLen := 7
	for i := 1; i <= 100; i++ {
		num := bin(i, binaryLen)
		numList := createPyList(t, 0)

		// create a list from binary numbers
		for _, n := range num {
			convertedNum := python3.PyLong_FromGoInt(int(n))
			assert.NotNil(t, convertedNum)
			assert.Zero(t, python3.PyList_Append(numList, convertedNum))
		}

		// the created list should be the same length as binary length
		assert.Equal(t, binaryLen, python3.PyList_Size(numList))

		list := createPyList(t, 0)
		assert.Zero(t, python3.PyList_Insert(list, 0, numList))

		// call method `test` with arguments and capture result. The result
		// should come back as a PyList of len == 4.
		output := obj.CallMethodObjArgs(callable, list)
		assert.True(t, python3.PyList_Check(output))
		assert.True(t, python3.PyList_CheckExact(output))
		assert.Equal(t, 4, python3.PyList_Size(output))

		resultList := make([]float32, 4)
		for j := 0; j < 4; j++ {
			// parse from PyList into Go list
			item := python3.PyList_GetItem(output, j)
			assert.NotNil(t, item)
			assert.Equal(t, python3.Float, item.Type())

			// parse the item from `PyFloat` to `float64`
			itemAsFloat := python3.PyFloat_AsDouble(item)
			assert.NotNil(t, itemAsFloat)

			// append to the result list
			resultList[j] = float32(itemAsFloat)
		}

		// check whether the results are correct
		assertFizzBuzz(t, resultList, i)

		list.DecRef()
		numList.DecRef()
	}

}

// bin will encode a number into binary and reverse it.
func bin(n int, num_digits int) []float32 {
	f := make([]float32, num_digits)
	for i := uint(0); i < uint(num_digits); i++ {
		f[i] = float32((n >> i) & 1)
	}
	return f[:]
}

// dec will return the value which is higher than 0.4 as the result.
func dec(b []float32) int {
	for i := 0; i < len(b); i++ {
		if b[i] > 0.4 {
			return i
		}
	}
	return -1
}

// assertFizzBuzz will decode the array of results and will determine whether
// the result is correct or not.
// The input should look like this: [0.912515, 0.05342, 0.23422, 0.4435].
// 0: its a number, e.g. 13, 16, 77.
// 1: its a Fizz!
// 2: its a Buzz!
// 3: its a FizzBuzz!
func assertFizzBuzz(t *testing.T, v []float32, i int) {
	decoded := dec(v)
	assert.Equal(t, decoded, fizzBuzz(i))
}

// fizzBuzz will take a number and return a corresponding fizzbuzz code.
func fizzBuzz(number int) int {
	if number%15 == 0 {
		return 3
	} else if number%5 == 0 {
		return 2
	} else if number%3 == 0 {
		return 1
	} else {
		return 0
	}
}

func createPyList(t *testing.T, len int) *python3.PyObject {
	pylist := python3.PyList_New(len)
	assert.True(t, python3.PyList_Check(pylist))
	assert.True(t, python3.PyList_CheckExact(pylist))
	return pylist
}
