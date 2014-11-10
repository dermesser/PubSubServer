package main

import (

)

type NullWriter struct {}

func (w *NullWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
