// +build !windows

package main

func run() {
	stop = make(chan struct{})
	signalProcess()
}
