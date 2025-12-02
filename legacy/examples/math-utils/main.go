package main

import (
	"fmt"
	"math"
)

// Add returns the sum of two integers
func Add(a, b int) int {
	return a + b
}

// Multiply returns the product of two integers
func Multiply(a, b int) int {
	return a * b
}

// IsPrime checks if a number is prime
func IsPrime(n int) bool {
	if n <= 1 {
		return false
	}
	if n <= 3 {
		return true
	}
	if n%2 == 0 || n%3 == 0 {
		return false
	}
	for i := 5; i*i <= n; i += 6 {
		if n%i == 0 || n%(i+2) == 0 {
			return false
		}
	}
	return true
}

// Factorial returns the factorial of n
func Factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * Factorial(n-1)
}

// Sqrt returns the square root using Newton's method
func Sqrt(x float64) float64 {
	return math.Sqrt(x)
}

func main() {
	fmt.Println("Math Utils Library")
	fmt.Println("==================")
	fmt.Printf("Add(5, 3) = %d\n", Add(5, 3))
	fmt.Printf("Multiply(4, 7) = %d\n", Multiply(4, 7))
	fmt.Printf("IsPrime(17) = %v\n", IsPrime(17))
	fmt.Printf("Factorial(5) = %d\n", Factorial(5))
	fmt.Printf("Sqrt(16) = %.2f\n", Sqrt(16))
}
