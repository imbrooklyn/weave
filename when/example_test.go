package when_test

import (
	"fmt"
	"strings"

	"github.com/imbrooklyn/weave/when"
)

func Example() {
	notReserved := func(value string) bool {
		return !strings.EqualFold(value, "all")
	}
	include := when.All(when.NotBlank, notReserved)

	fmt.Println(include("active"))
	fmt.Println(include("   "))
	fmt.Println(include("ALL"))

	// Output:
	// true
	// false
	// false
}

func ExampleNotEmpty() {
	type identifiers []int64

	var empty identifiers
	selected := identifiers{10, 20}
	fmt.Println(when.NotEmpty(empty))
	fmt.Println(when.NotEmpty(selected))

	// Output:
	// false
	// true
}
