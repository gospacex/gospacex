package microservice

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMicroserviceDDDGenerator(t *testing.T) {
	g := NewMicroserviceDDDGenerator("/tmp/test", "/tmp/output")
	assert.NotNil(t, g)
}
