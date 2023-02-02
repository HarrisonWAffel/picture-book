package pkg

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestRetag(t *testing.T) {
	image := "rancher/hardened-flannel:v0.13.0-rancher1-build20210223"
	repository := "rancher"
	host := "my-registry.space"

	assert.Equal(t, ReTag(image, host, repository), "my-registry.space/rancher/hardened-flannel:v0.13.0-rancher1-build20210223")
	assert.Equal(t, ReTag(image, host, "test-image"), "my-registry.space/test-image/discovery-server:latest")
	assert.Equal(t, ReTag(image, host, ""), "my-registry.space/discovery-server:latest")
}
