package mqtt

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/DataDog/go-python3"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Setenv("PYTHONPATH", "./")
	os.Exit(m.Run())
}

func TestPythonInitialization(t *testing.T) {
	randomString := "random"

	python3.Py_Initialize()
	defer python3.Py_Finalize()

	// get the module named interpreter
	obj := python3.PyImport_ImportModule("interpreter")
	assert.NotNil(t, obj)

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

func TestPythonCallMethod(t *testing.T) {
	python3.Py_Initialize()
	defer python3.Py_Finalize()

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

func TestSaveModel(t *testing.T) {
	modelData := []byte{0x1c, 0x00, 0x00, 0x00, 0x54, 0x46, 0x4c, 0x33}
	hermes := &hermes{}
	mac := "testMac"

	// save test model data
	hermes.saveModel(modelData, mac)

	// confirm that the model was written to
	savedModelName := fmt.Sprintf("%s/model_%s.tflite", modelsDir, mac)

	data, err := ioutil.ReadFile(savedModelName)
	if err != nil {
		t.Errorf("Failed to read file = %s", err)
	}

	if bytes.Compare(modelData, data) != 0 {
		t.Errorf("%s != %s", modelData, data)
	}
}

func TestGetCanSend(t *testing.T) {
	hermes := hermes{}
	testData := []struct {
		mac             string
		canSend         bool
		expectedCanSend bool
		time            time.Duration
	}{
		// should finish instantly - so can send
		{"AA:BB:CC:DD:EE:FF", true, true, time.Nanosecond},
		// should 'never' finish (in test) - so cannot send
		{"AA:BB:CC:DD:EE:FB", true, false, time.Minute * 4},
		// should send, as a timer for it doesn't exist
		{"AA:BB:CC:DD:EE:FA", false, true, time.Second},
	}

	hermes.Initialize()

	count := 0
	for _, data := range testData {
		if data.canSend {
			hermes.canSend[data.mac] = data.canSend
			hermes.sendTicker[data.mac] = time.NewTicker(data.time)
			count++
		}
	}
	assert.Len(t, hermes.canSend, count, "canSend wrong len")
	assert.Len(t, hermes.sendTicker, count, "sendTicker wrong len")

	for _, data := range testData {
		assert.Equal(t, data.expectedCanSend, hermes.GetCanSend(data.mac))
	}
}

func TestParseMac(t *testing.T) {
	var topicTests = []struct {
		topic  string
		result string
	}{
		{"hermes/global/AA:BB:CC:DD:EE:FF/model/receive", "AA:BB:CC:DD:EE:FF"},
		{"hermes/AA:BB:CC:DD:EE:FF/global/receive/+", "AA:BB:CC:DD:EE:FF"},
		{"randomlongtext:randomverylongtext/global/send", ""},
		{"hermes/AA:BB:CC:DD:EE/global/send", ""},
		{"", ""},
	}

	for _, test := range topicTests {
		assert.Equal(t, test.result, parseTopicMac(test.topic), "Result is invalid")
	}
}
