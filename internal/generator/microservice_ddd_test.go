package generator

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewMicroserviceDDDGenerator(t *testing.T) {
	g := NewMicroserviceDDDGenerator("/tmp/test", "/tmp/output")
	assert.NotNil(t, g)
}
