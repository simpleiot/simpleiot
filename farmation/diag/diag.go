package diag

import (
	"fmt"
	"runtime"
	"strings"
)

// Diag is an interface that all diags implement
type Diag interface {
	Run() error
	String() string
}

var diags = []Diag{}

// Register is used to register new tests
func Register(d Diag) {
	diags = append(diags, d)
}

// List lists all diags
func List() {
	fmt.Println("Available diagnostic tests:")
	for _, d := range diags {
		fmt.Println(" ", d)
	}
}

func stopMainApp() {
	//exec.Command("/etc/init.d/iswatchdog", "stop").Run()
	//exec.Command("/etc/init.d/isapp", "stop").Run()
}

// RunSingle runs a single diag test
func RunSingle(test string) bool {
	stopMainApp()

	if runtime.GOARCH != "arm" {
		fmt.Println("Error, dianostics can only be run on IS platform")
		return false
	}

	for _, d := range diags {
		if d.String() == test {
			err := d.Run()
			if err == nil {
				fmt.Println(d, "(Passed)")
				return true
			}

			fmt.Println(d, "(Failed)", err)
			return false
		}
	}

	fmt.Println("Error, could not find test", test)
	return false
}

// ListTests lists all available tests
func ListTests() {
	fmt.Println("Available tests:")
	for _, d := range diags {
		fmt.Println(" ", d)
	}
}

// Run runs all diags
func Run() {
	stopMainApp()
	if runtime.GOARCH != "arm" {
		fmt.Println("Error, dianostics can only be run on IS platform")
		return
	}

	fmt.Println("Running diagnostics")
	var failedCount = 0
	for _, d := range diags {
		err := d.Run()
		if err == nil {
			fmt.Println(d, "(Passed)")
		} else {
			fmt.Println(d, "(Failed): ", err)
			failedCount++
		}
	}

	if failedCount == 0 {
		fmt.Println("All tests passed")
	} else {
		fmt.Println(failedCount, "tests failed")
	}
}

// GetEnter waits for user to press enter
func GetEnter(prompt string) string {
	fmt.Println(prompt + " and press enter")
	var input string
	fmt.Scanln(&input)
	return input
}

// GetInput returns user input
func GetInput(prompt string) bool {
	var input string

	if prompt != "" {
		fmt.Println(prompt + " (y/n)?")
	}

	_, err := fmt.Scanln(&input)
	if err != nil {
		fmt.Println("Error reading input from user", err)
		return false
	}

	input = strings.Trim(input, " ")
	if input == "y" || input == "Y" {
		return true
	}

	return false
}
