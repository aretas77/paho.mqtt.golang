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

const (
	defaultNoSendInterval = time.Second * 1
)

func TestHermesCheckNewInterval(t *testing.T) {
	hermes := &hermes{}
	mac := "00:00:00:00:00:00"
	hermes.Initialize()

	assert.Equal(t, 0, hermes.counter[mac])

	hermes.counter[mac] = 4
	hermes.checkNeedNewInterval(nil, mac)
	assert.Equal(t, 0, hermes.counter[mac])
}

func TestHermesRequestNewInterval(t *testing.T) {
	clientOptions := NewClientOptions()
	mac := "00:00:00:00:00:00"

	// Override some default values
	clientOptions.UseHermes = true
	clientOptions.SetClientID("device-test")
	clientOptions.AddBroker("tcp://172.18.0.3:1883")
	clientOptions.SetUsername("devices")
	clientOptions.SetPassword("secretkey987")
	clientOptions.SetCleanSession(true)
	clientOptions.SetProtocolVersion(3)
	client := NewClient(clientOptions)

	token := client.Connect()
	if token.WaitTimeout(time.Minute * 2); token.Error() != nil {
		t.Fatalf("client failed to connect: %s", token.Error())
	}
	assert.True(t, client.IsConnected())

	// grab the Hermes reader
	hermes := client.HermesReader()

	// get the current send interval
	old_interval := hermes.CallGetCurrentSendInterval(mac)
	assert.Equal(t, old_interval, defaultNoSendInterval)

	// send a request to the Hades server
	if err := hermes.CallRequestNewInterval(client, mac); err != nil {
		t.Fatalf("hermes failed to request a new interval: %s", err)
	}

	// wait for response
	time.Sleep(time.Second * 2)

	// check if interval updated
	current_interval := hermes.CallGetCurrentSendInterval(mac)
	assert.NotEqual(t, current_interval, old_interval)
}

func TestHermesRequestNewModel(t *testing.T) {
	clientOptions := NewClientOptions()
	mac := "00:00:00:00:00:00"

	path := fmt.Sprintf("models/model_%s.tflite", mac)
	os.Remove(path)

	// Override some default values
	clientOptions.UseHermes = true
	clientOptions.SetClientID("device-test")
	clientOptions.AddBroker("tcp://172.18.0.3:1883")
	clientOptions.SetUsername("devices")
	clientOptions.SetPassword("secretkey987")
	clientOptions.SetCleanSession(true)
	clientOptions.SetProtocolVersion(3)
	client := NewClient(clientOptions)

	token := client.Connect()
	if token.WaitTimeout(time.Minute * 2); token.Error() != nil {
		t.Fatalf("client failed to connect: %s", token.Error())
	}
	assert.True(t, client.IsConnected())

	// send a request to the server
	hermes := client.HermesReader()
	if err := hermes.CallRequestNewModel(client, mac); err != nil {
		t.Fatalf("hermes failed to request a new model: %s", err)
	}

	// wait for the file
	time.Sleep(time.Second * 2)

	// check if file exists
	_, err := os.Stat(path)
	assert.False(t, os.IsNotExist(err))

	hermes.Finalize()
}

func TestHermesGetSetSendInterval(t *testing.T) {
	clientOptions := NewClientOptions()
	mac := "00:00:00:00:00:00"

	// Override some default values
	clientOptions.UseHermes = true
	clientOptions.SetClientID("device-test")
	clientOptions.AddBroker("tcp://172.18.0.3:1883")
	clientOptions.SetUsername("devices")
	clientOptions.SetPassword("secretkey987")
	clientOptions.SetCleanSession(true)
	clientOptions.SetProtocolVersion(3)
	client := NewClient(clientOptions)

	token := client.Connect()
	if token.WaitTimeout(time.Minute * 2); token.Error() != nil {
		t.Fatalf("client failed to connect: %s", token.Error())
	}
	assert.True(t, client.IsConnected())

	// Check the default value
	hermes := client.HermesReader()
	current_interval := hermes.CallGetCurrentSendInterval(mac)
	assert.Equal(t, current_interval, defaultNoSendInterval)

	// Check the custom set interval - wait for a bit
	// XXX: implement token system for sendTimer
	hermes.CallSetSendInterval(mac, time.Minute*5)
	time.Sleep(time.Second * 2)
	current_interval = hermes.CallGetCurrentSendInterval(mac)
	assert.Equal(t, current_interval, time.Minute*5)
}

func TestHermesSaveModel(t *testing.T) {
	modelData := []byte{0x1c, 0x00, 0x00, 0x00, 0x54, 0x46, 0x4c, 0x33}
	hermes := &hermes{}
	mac := "00:00:00:00:00:00"

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

func TestHermesGetCanSend(t *testing.T) {
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

	time.Sleep(time.Second)
	for _, data := range testData {
		assert.Equal(t, data.expectedCanSend, hermes.GetCanSend(nil, data.mac))
	}
}

func TestHermesParseMac(t *testing.T) {
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
