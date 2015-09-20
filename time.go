package main

import (

)

type Timer struct {
	dt int
	cancel int
	c chan int
	next *Timer
}