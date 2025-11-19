//go:build ignore

package main

//go:generate go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/sendpulse/main.go -o docs --parseDependency --parseInternal --parseDepth 3