# Math Utils Example

A collection of mathematical utility functions demonstrating a more complex LibreSeed package.

## Features

- Addition and multiplication
- Prime number checking
- Factorial calculation
- Square root computation

## Usage

```bash
go run main.go
```

## As a Library

```go
import "github.com/yourorg/math-utils"

result := mathutils.Add(5, 3)
isPrime := mathutils.IsPrime(17)
```

## Building with LibreSeed

```bash
packager create -key mykey.private -dir . -out ./dist -name math-utils -version 2.0.0
```
