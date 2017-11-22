package featuretoggle_test

import (
	"fmt"
	"testing"

	unleash "github.com/Unleash/unleash-client-go"
	unleashcontext "github.com/Unleash/unleash-client-go/context"
	"github.com/fabric8-services/fabric8-wit/featuretoggle"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnabledFeatures(t *testing.T) {
	resource.Require(t, resource.Toggles)

	t.Run("feature enabled for beta user", func(t *testing.T) {
		// given
		client, err := unleash.NewClient(
			unleash.WithAppName("Fabric8"),
			unleash.WithUrl("http://localhost:4242/api/"),
			unleash.WithStrategies(&featuretoggle.RolloutByGroupIDStrategy{}),
		)
		defer client.Close()
		// wait until client did perform a data sync
		<-client.Ready()
		require.Nil(t, err)
		// when
		ctx := &unleashcontext.Context{
			Properties: map[string]string{
				"groupID": "BETA",
			},
		}
		features := client.GetEnabledFeatures(ctx)
		// then
		fmt.Printf("Enabled features: %v\n", features)
		assert.NotEmpty(t, features)
	})

}
