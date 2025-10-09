package async_test

import (
	"testing"

	"github.com/rubengp99/go-async"
	"github.com/stretchr/testify/assert"
)

func TestDrainer(t *testing.T) {

	drainer := async.NewDrainer[typeA]()
	drainer.Send(typeA{value: "1"})
	drainer.Send(typeA{value: "2"})
	drainer.Send(typeA{value: "3"})

	results := drainer.DrainAndShutDown()

	t.Run("results as expected", func(t *testing.T) {
		assert.Equal(t, 3, len(results))
	})
}
