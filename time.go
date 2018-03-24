package main

type Timer struct {
	dt     int
	cancel int
	c      chan int
	next   *Timer
}
