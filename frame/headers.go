package frame


type HeaderPayload struct {

}

func HeaderPayloadFromFrame(f *Frame) (*HeaderPayload, error) {
	return nil, nil

}

func NewHeaderPayload() *HeaderPayload {
	return &HeaderPayload{}
}

func (hp *HeaderPayload) ToFrame(fr *Frame) {
}
