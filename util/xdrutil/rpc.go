package xdrutil

func CallRpcVoid[I any](f func(I), in I) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	f(in)
	return nil
}

func CallRpc[I any, O any](f func(I) O, in I) (out O, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	out = f(in)
	return out, nil
}
