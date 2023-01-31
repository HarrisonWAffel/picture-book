package pkg

import (
	"github.com/docker/docker/api/types"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestRetag(t *testing.T) {
	image := "harrisonwaffel/discovery-server:latest"
	host := "haffel1.cp-dev.rancher.space"

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
