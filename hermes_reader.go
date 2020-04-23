package mqtt

// ClientHermesReader provides a wrapper interface for reading hermes struct
// and calling its methods.
type ClientHermesReader struct {
	hermes *hermes
}

// CallRequestNewModel ...
func (r *ClientHermesReader) CallRequestNewModel(c Client, mac string) error {
	return r.hermes.RequestNewModel(c, mac)
}

// CallPingHades will call internal hermes PingHades method.
func (r *ClientHermesReader) CallPingHades(c Client, mac string) bool {
	return r.hermes.PingHades(c, mac)
}

// GetHandlers will return a copy of subscribed handlers.
func (r *ClientHermesReader) GetHandlers() []TopicHandler {
	h := r.hermes.handlers
	return h
}
