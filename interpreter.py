#!/usr/bin/python3.7

"""
    This is a wrapper for TensorFlow Lite interpreter and is used by the
    paho.mqtt.golang library to extract the values from the given model.
"""

import sys
import os
from importlib import import_module

# check for required libraries and try to import them.
libnames = [("numpy", "np"), ("tensorflow", "tf")]
for (name, short) in libnames:
    try:
        lib = import_module(name)
    except ImportError:
        print(sys.exc_info())
    else:
        globals()[short] = lib


class Interpreter:
    """Interpreter class is responsible for loading the TensorFlow Lite model
    with given values and getting some kind of output from it and returning to
    the caller.
    """
    models = "models"

    def __init__(self, mac, path):
        self.mac = mac
        self.modelname = "model_" + mac + ".tflite"
        self.path = path
        self.models_dir = os.path.join(path, self.models)

        # print(self.check())

    def init_interpreter(self, path):
        """init_interpreter will load a TensorFlow Lite model and allocate the
        tensors.
        """
        interpreter = tf.lite.Interpreter(model_path=path)
        interpreter.allocate_tensors()
        return interpreter

    # check should determine whether a model exists or not.
    def check(self):
        return os.path.exists(self.models_dir + "/" + self.modelname)

    def test_inference(self, number):
        """test_inference runs a 'test run' of TensorFlow Lite interpreter with
        a default model for a FizzBuzz problem. The goal of this method is to
        test whether TensorFlow Lite works as expected and its possible to do
        inference.

        Suppose we want to infere a 4 with binary length of 7:
            bin(4) = 0000100
            reverse(0000100) = 0010000

            bin(16) = 0010000
            reverse(001000) = 0000100

        Args:
            number: is a number in binary form in reverse.
        """
        fizzbuzz = os.path.join(self.models, "fizzbuzz_model.tflite")
        interpreter = self.init_interpreter(fizzbuzz)

        # Get input and output tensors.
        input_details = interpreter.get_input_details()
        output_details = interpreter.get_output_details()

        # For fizzbuzz_model an input should consist of [1 7] array.
        self.assert_shape(input_details[0], [1, 7])
        # And output of [1 4] array.
        self.assert_shape(output_details[0], [1, 4])

        # Test model on a user input data
        input_shape = input_details[0]['shape']

        # Setup input data for the model
        input_data = np.array(number, dtype=np.float32)
        interpreter.set_tensor(input_details[0]['index'], input_data)

        interpreter.invoke()

        # The function `get_tensor()` returns a copy of the tensor data.
        # Use `tensor()` in order to get a pointer to the tensor.
        output_data = interpreter.get_tensor(output_details[0]['index'])
        return output_data[0].tolist()

    def assert_shape(self, x, shape: list):
        """assert_shape will be used to check whether an input or output shape
        corresponds to the expected shape. Example:
            assert_shape(input_shape, [1, 7, None, 7])
        """
        assert len(x['shape']) == len(shape), (x['shape'], shape)
        for _a, _b in zip(x['shape'], shape):
            if isinstance(_b, int):
                assert _a == _b, (x['shape'], shape)


def start(device_mac):
    # We need to get current directory name through environment variable.
    # TODO: mmmm? this whole interpreter stuff shouldn't even be here.
    current_dir = os.environ["PYTHONPATH"]
    interpreter = Interpreter(device_mac, current_dir)
    return


"""
Methods used primarily for TensorFlow Lite testing.
"""


def test(test_input):
    # test is used for unit testing to test python3 integration with Go.
    return test_input


def test_inference(number):
    interpreter = Interpreter("", "")
    return interpreter.test_inference(number)


start("AA:BB:CC:DD:EE:FF")
