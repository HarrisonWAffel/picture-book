package pkg

import (
	"github.com/docker/docker/api/types"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestRetag(t *testing.T) {
	image := "rancher/hardened-flannel:v0.13.0-rancher1-build20210223"
	repository := "rancher"
	host := "haffel1.cp-dev.rancher.space"

	assert.Equal(t, ReTag(image, host, repository), "haffel1.cp-dev.rancher.space/rancher/hardened-flannel:v0.13.0-rancher1-build20210223")
	assert.Equal(t, ReTag(image, host, "test-image"), "haffel1.cp-dev.rancher.space/test-image/discovery-server:latest")
	assert.Equal(t, ReTag(image, host, ""), "haffel1.cp-dev.rancher.space/discovery-server:latest")
}

func TestAUth(t *testing.T) {
	cfg := types.AuthConfig{
		Username:      "user",
		Password:      "password1234",
		ServerAddress: "haffel2.cp-dev.drancher.space",
	}

	t.Log(BuildEncodedAuthConfig(cfg))
}
