package async

// Drainable is an interfacee that wraps a drainable channel/async output functionality
type Drainable interface {
	ShutDown()
}

// Drain wraps a channel as an output drainer
type Drain[T any] chan T

// Drain drains all values from the channel and returns them in a slice
func (d Drain[T]) Drain() []T {
	var result []T
	// Loop to receive from the channel until it is closed
L:
	for {
		select {
		case v, ok := <-d:
			if !ok { //ch is closed //immediately return err
				break L
			}

			result = append(result, v)
		default: //all other case not-ready: means nothing in ch for now
			break L
		}
	}
	return result
}
