package main

import (
	"errors"
	"strconv"
)

var ErrZero = errors.New("zero is not allowed")

func fizzBuzz(n int) (string, error) {
	if n == 0 {
		return "", ErrZero
	}

	divisibleBy3 := n%3 == 0
	divisibleBy5 := n%5 == 0

	switch {
	case divisibleBy3 && divisibleBy5:
		return "FizzBuzz", nil
	case divisibleBy3:
		return "Fizz", nil
	case divisibleBy5:
		return "Buzz", nil
	default:
		return strconv.Itoa(n), nil
	}
}
