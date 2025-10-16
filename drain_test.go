package gopool_test

import (
	"fmt"
	"testing"

	gopool "github.com/rubengp99/go-pool"
	"github.com/stretchr/testify/assert"
)

func TestDrainer(t *testing.T) {

	drainer := gopool.NewDrainer[typeA]()
	drainer.Send(typeA{value: "1"})
	drainer.Send(typeA{value: "2"})
	drainer.Send(typeA{value: "3"})

	results := drainer.Drain()

	t.Run("results as expected", func(t *testing.T) {
		assert.Equal(t, 3, len(results))
	})
}

func BenchmarkDrainerVsAppend(b *testing.B) {
	b.Run("drainer", func(b *testing.B) {
		drainer := gopool.NewDrainer[typeA]()
		for i := 0; i < b.N; i++ {
			drainer.Send(typeA{value: fmt.Sprint(i)})
		}

	})

	b.Run("slice", func(b *testing.B) {
		slice := make([]typeA, 0)

		for i := 0; i < b.N; i++ {
			slice = append(slice, typeA{value: fmt.Sprint(i)})
		}
	})
}
