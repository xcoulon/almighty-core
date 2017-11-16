package feature_toggle_test

import (
	"context"
	"testing"

	unleash "github.com/Unleash/unleash-client-go"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnabledFeatures(t *testing.T) {
	resource.Require(t, resource.Toggles)

	t.Run("feature enabled for beta user", func(t *testing.T) {
		// given
		client, err := unleash.NewClient(
			unleash.WithAppName("fabric8"),
			unleash.WithUrl("http://localhost:4242/api/"),
			unleash.WithStrategies(&RolloutByGroupIDStrategy{}),
		)
		require.Nil(t, err)
		// when
		ctx := &context.Context{
			Properties: map[string]string{
				"groupID": "BETA",
			},
		}
		features := client.GetEnabledFeatures(ctx)
		// then
		assert.NotEmpty(t, features)
	})

}
