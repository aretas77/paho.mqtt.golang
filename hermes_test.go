package mqtt

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Setenv("PYTHONPATH", "./")
	os.Exit(m.Run())
}

func TestRequestNewModel(t *testing.T) {
	clientOptions := NewClientOptions()

	// Override some default values
	clientOptions.UseHermes = true
	clientOptions.SetClientID("device-test")
	clientOptions.AddBroker("tcp://172.18.0.3:1883")
	clientOptions.SetUsername("devices")
	clientOptions.SetPassword("secretkey987")
	clientOptions.SetCleanSession(true)
	client := NewClient(clientOptions)

	token := client.Connect()
	if token.WaitTimeout(30); token.Error() != nil {
		t.Fatalf("client failed to connect: %s", token.Error())
	}
	assert.True(t, client.IsConnected())

	// At this point we have a connection - send a request for a model.

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
