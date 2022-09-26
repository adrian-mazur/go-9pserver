package main

func min[K uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64](a K, b K) K {
	if a < b {
		return a
	}
	return b
}
