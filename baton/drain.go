package baton

func drain[T any](ch <-chan T, fallback T) (last T) {
	last = fallback
	for {
		select {
		case msg, ok := <-ch:
			if ok {
				last = msg
			} else {
				return
			}
		default:
			return
		}
	}
}

func waitAndDrain[T any](ch <-chan T) T {
	return drain(ch, <-ch)
}
