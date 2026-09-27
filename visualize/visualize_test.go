package visualize_test

import (
	"testing"

	"github.com/danceable/container/visualize"
	"github.com/stretchr/testify/assert"
)

func TestWithRenderer(t *testing.T) {
	t.Parallel()

	o := visualize.DefaultOptions(visualize.DOT{})
	assert.Equal(t, visualize.DOT{}, o.Renderer)

	visualize.WithRenderer(visualize.ASCII{Color: true})(o)
	assert.Equal(t, visualize.ASCII{Color: true}, o.Renderer)

	visualize.WithRenderer(nil)(o)
	assert.Equal(t, visualize.ASCII{Color: true}, o.Renderer, "a nil renderer keeps the one set")
}
