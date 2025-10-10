package gopool_test

import (
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
