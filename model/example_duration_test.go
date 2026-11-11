package model_test

import (
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

func ExampleDuration() {
	// Create durations using familiar Go syntax
	d1 := model.MustParseDuration("30s")
	d2 := model.MustParseDuration("1h30m")
	d3 := model.MustParseDuration("2h")

	fmt.Println("30 seconds:", d1.String())
	fmt.Println("1.5 hours:", d2.String())
	fmt.Println("2 hours:", d3.String())

	// Output:
	// 30 seconds: 30s
	// 1.5 hours: 1h30m0s
	// 2 hours: 2h0m0s
}

func ExampleDuration_withTask() {
	// Use optional.Of to set optional fields
	task := &model.Task{
		TaskType:  optional.Of("bash"),
		Arguments: []string{"-c", "echo hello"},
		Timeout:   optional.Of(model.MustParseDuration("5m")), // 5 minutes in Go format
	}

	fmt.Printf("Task timeout: %v\n", task.Timeout.Or(0).String())

	// Output:
	// Task timeout: 5m0s
}

func ExampleParseDuration() {
	d, err := model.ParseDuration("1h30m45s")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Duration: %v\n", d.String())

	// Output:
	// Duration: 1h30m45s
}
