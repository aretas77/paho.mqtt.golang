package mqtt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/DataDog/go-python3"
)

const (
	// modelsDir will be used to receive/read models.
	modelsDir = "./models"
	// hermesPrefix will be used to match topics related to internal library.
	hermesPrefix = "hermes"
	// hadesPrefix will be used to match topics related to Hades observer
	// is mainly used for publishing.
	hadesPrefix = "hades"
	// TimerSendInterval is used for setting the Send interval for the Timer.
	TimerSendInterval = "timerSendInterval"
	// TimerReceiveInterval is used for setting the Receive interval for the Timer.
	TimerReceiveInterval = "timerRecvInterval"
)

// hermes is the main struct for hermes subsystem in the library. It controls
// when the library can send by managing timers, manages models and holds the
// reference to the interpreter.
type hermes struct {
	interpreter *python3.PyObject

	// not used - should only be used for one device to be aware of its power.
	batteryLeftMah  float32
	totalBatteryMah float32
	lastModelUpdate time.Time
	initialModel    bool

	currentSendInterval map[string]time.Duration
	sendTicker          map[string]*time.Ticker
	canSend             map[string]bool
	rwMutex             sync.RWMutex

	handlers []TopicHandler

	// these are control channels which are used to control the timer.
	setTimer   chan *Timer
	resetTimer chan string

	serverAlive bool
	lastCheck   time.Time
}

// timer struct will be used to send the data to set the timer durations for
// various library functions.
type Timer struct {
	duration  time.Duration
	timerType string
	mac       string
}

// TopicHandler is used for easier topic subscription - it will contain the
// topic, the topics' QoS and handler to process messages on the given Topic.
type TopicHandler struct {
	Topic   string
	QoS     byte
	Handler func(c Client, msg Message)
}

// RequestModelPayload contains data for the hermes to request a new model.
type RequestModelPayload struct {
	MAC             string    `json:"mac"`
	LastModelUpdate time.Time `json:"last_model_update"`
	Initial         bool      `json:"initial"`
}

// SendIntervalPayload contains data for the hermes to process the received
// send interval change request.
type SendIntervalPayload struct {
	MAC          string `json:"mac"`
	SendInterval int    `json:"send_interval"`
}

// Initialize will initialize the hermes structure which will be responsible
// for managing the publishing of new messages.
func (h *hermes) Initialize() {
	python3.Py_Initialize()

	// a common channel for setting new values for a Timer.
	h.setTimer = make(chan *Timer)

	// for each device we have a unique canSend flag and a unique timer.
	h.resetTimer = make(chan string)
	h.canSend = make(map[string]bool)
	h.sendTicker = make(map[string]*time.Ticker)
	h.currentSendInterval = make(map[string]time.Duration)

	h.initialModel = true
	h.interpreter = python3.PyImport_ImportModule("interpreter")
	if h.interpreter == nil {
		CRITICAL.Println(HER, "Initialize() failed to import interpreter")
	}

	// initialize the topics with their handlers for hermes
	h.handlers = []TopicHandler{
		{"node/+/+/hades/model/receive", 1, h.HandleReceiveModel},
		{"node/+/+/hades/interval/receive", 1, h.HandleReceiveInterval},
	}
}

// SaveModel will receive a model in bytes and will save it in the given models
// directory.
func (h *hermes) saveModel(model []byte, mac string) {
	modelName := fmt.Sprintf("%s/model_%s.tflite", modelsDir, mac)
	err := ioutil.WriteFile(modelName, model, 0644)
	if err != nil {
		ERROR.Println(err)
	}
}

// SetSendInterval will set the given send interval as the current send
// interval for the given device.
func (h *hermes) SetSendInterval(mac string, interval time.Duration) {
	h.setTimer <- &Timer{
		duration:  interval,
		timerType: TimerSendInterval,
		mac:       mac,
	}
}

// GetHandlers will return the initialized handlers to the caller with prepended
// hermesPrefix.
func (h *hermes) GetHandlers() []TopicHandler {
	for i, entry := range h.handlers {
		h.handlers[i].Topic = fmt.Sprintf("%s/%s", hermesPrefix, entry.Topic)
	}

	return h.handlers
}

// GetCurrentSendInterval will return the currently used send interval for the
// appropriate MAC device.
func (h *hermes) GetCurrentSendInterval(mac string) time.Duration {
	h.rwMutex.Lock()
	defer h.rwMutex.Unlock()

	// no rules are set - allow send
	if h.currentSendInterval[mac] == 0 {
		return time.Second * 1
	}

	return h.currentSendInterval[mac]
}

// GetCanSend will return whether the timer allows to send the data for the
// library. It will wait for the ticker to finish and set the canSend flag or
// set canSend as false by default (if ticker hasn't ticked).
func (h *hermes) GetCanSend(mac string) bool {
	h.rwMutex.Lock()
	defer h.rwMutex.Unlock()

	// no rules are set - allow send
	if len(h.canSend) == 0 || h.sendTicker[mac] == nil {
		return true
	}

	select {
	case <-h.sendTicker[mac].C:
		h.canSend[mac] = true
	default:
		h.canSend[mac] = false
	}

	return h.canSend[mac]
}

// RequestNewModel should send a request for a model to the Hades server. A handle
// should receive the requested model.
func (h *hermes) RequestNewModel(c Client, mac string) error {
	payload := RequestModelPayload{
		MAC:             mac,
		LastModelUpdate: h.lastModelUpdate,
		Initial:         h.initialModel,
	}

	resp, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// XXX: We currently do not know when Hades will be ready - maybe it won't
	// send a model?
	requestTopic := fmt.Sprintf("%s/global/%s/model/request", hadesPrefix, mac)
	token := c.Publish(requestTopic, 1, false, resp)
	if token.Error() != nil {
		WARN.Println(HER, "request for model has failed")
		return token.Error()
	}

	return nil
}

func (h *hermes) RequestNewInterval(c Client, mac string) error {
	payload := RequestModelPayload{
		MAC:             mac,
		LastModelUpdate: h.lastModelUpdate,
		Initial:         h.initialModel,
	}

	resp, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	requestTopic := fmt.Sprintf("%s/global/%s/interval/request", hadesPrefix, mac)
	token := c.Publish(requestTopic, 1, false, resp)
	if token.Error() != nil {
		WARN.Println(HER, "request for send interval has failed")
		return token.Error()
	}

	return nil
}

func (h *hermes) IsConnectedHades() bool {
	h.rwMutex.RLock()
	defer h.rwMutex.RUnlock()
	return h.serverAlive
}

// HandleReceiveModel is called when a model was received. Interpreter should
// called to parse the received values.
// TODO: implement interpreter call.
func (h *hermes) HandleReceiveModel(c Client, msg Message) {
	// retrieve MAC address so we should know for whom to set the timer.
	mac := parseTopicMac(msg.Topic())
	h.saveModel(msg.Payload(), mac)

	// mark that initial model is received
	h.initialModel = false

	// read the values from the model and send to the ticker
	h.setTimer <- &Timer{
		duration:  time.Second * 10,
		timerType: TimerSendInterval,
		mac:       mac,
	}
}

// HandleReceiveInterval is called when a new send interval was received from
// a server.
func (h *hermes) HandleReceiveInterval(c Client, msg Message) {
	mac := parseTopicMac(msg.Topic())

	payload := SendIntervalPayload{}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		WARN.Println(HER, "failed to parse received interval")
	}

	if payload.SendInterval == 0 || mac == "" {
		WARN.Println(HER, "received invalid send interval = ", payload.SendInterval)
		return
	}

	WARN.Println(HER, "received new interval")
	h.setTimer <- &Timer{
		duration:  time.Minute * time.Duration(payload.SendInterval),
		timerType: TimerSendInterval,
		mac:       mac,
	}
}

// Reset will reset the Hermes framework.
func (h *hermes) Reset() {
	python3.Py_Finalize()
}

// ResetCanSend is used to reset the flag which indicates that Publish is
// allowed for a device with a specific MAC address.
func (h *hermes) ResetCanSend(mac string) {
	if mac != "" {
		h.resetTimer <- mac
	}
}

// sendTimer will keep track of time for when a publish message is allowed.
//	* While timer is active, all messages received from the client will be discarded.
//	* The timer will toggle a flag for a client which will indicate whether sending
//	  is allowed or not.
//	* The timer will handle the setting of a new value for the mac <> ticker.
func (h *hermes) sendTimer(c *client) {
	defer c.workers.Done()
	defer func() {
		h.rwMutex.Lock()
		defer h.rwMutex.Unlock()
		for _, value := range h.sendTicker {
			value.Stop()
		}
	}()

	for {
		select {
		case newTime := <-h.setTimer:
			mac := newTime.mac

			WARN.Printf("%s received set %s event (MAC = %s)", HER,
				newTime.timerType, mac)
			// set a new interval for sending
			if newTime.timerType == TimerSendInterval {
				h.rwMutex.Lock()

				// clean resources
				if h.sendTicker[mac] != nil {
					h.sendTicker[mac].Stop()
				}

				// when initiating a new ticker - we disable sending.
				h.currentSendInterval[mac] = newTime.duration
				h.sendTicker[mac] = time.NewTicker(newTime.duration)
				h.canSend[mac] = false

				h.rwMutex.Unlock()
			}
		case mac := <-h.resetTimer:
			// when receiving from resetTimer channel, we restart the Timer
			// of a given mac address. Timer will be reset to its initial state.
			h.rwMutex.Lock()
			h.canSend[mac] = false
			h.sendTicker[mac].Stop()
			h.sendTicker[mac] = time.NewTicker(h.currentSendInterval[mac])
			h.rwMutex.Unlock()
		}
	}
}
