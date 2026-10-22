package main

import "slices"

func rotateRunes(s string, shift int) string {
	runes := []rune(s)
	runeCount := len(runes)

	if runeCount == 0 {
		return s
	}

	leftShift := shift % runeCount
	if leftShift < 0 {
		leftShift += runeCount
	}

	return string(slices.Concat(runes[leftShift:], runes[:leftShift]))
}
