package mqtt

import "time"

// ClientHermesReader provides a wrapper interface for reading hermes struct
// and calling its methods.
type ClientHermesReader struct {
	hermes *hermes
}

// CallRequestNewModel ...
func (r *ClientHermesReader) CallRequestNewModel(c Client, mac string) error {
	return r.hermes.RequestNewModel(c, mac)
}

func (r *ClientHermesReader) CallRequestNewInterval(c Client, mac string) error {
	return r.hermes.RequestNewInterval(c, mac)
}

func (r *ClientHermesReader) CallSetSendInterval(mac string, interval time.Duration) {
	r.hermes.SetSendInterval(mac, interval)
}

// CallGetCurrentSendInterval ...
func (r *ClientHermesReader) CallGetCurrentSendInterval(mac string) time.Duration {
	return r.hermes.GetCurrentSendInterval(mac)
}

// GetHandlers will return a copy of subscribed handlers.
func (r *ClientHermesReader) GetHandlers() []TopicHandler {
	h := r.hermes.handlers
	return h
}
