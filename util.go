package main

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func region(a, b int) int {
	if a < b {
		return -1
	}
	if a == b {
		return 0
	}
	return 1
}