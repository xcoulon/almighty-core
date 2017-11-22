package featuretoggle

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"

	unleash "github.com/Unleash/unleash-client-go"
	unleashcontext "github.com/Unleash/unleash-client-go/context"
	"github.com/fabric8-services/fabric8-wit/account"
)

// ToggleClient the toggle client
type ToggleClient struct {
	unleashClient unleash.Client
}

// ToggleServiceConfiguration the configuration to the Toggle service
type ToggleServiceConfiguration interface {
	// GetToggleServiceAppName() string
	GetToggleServiceAPIURL() string
}

// NewFeatureToggleClient returns a new client to the toggle feature service
func NewFeatureToggleClient(ctx context.Context, config ToggleServiceConfiguration) (*ToggleClient, error) {
	client, err := unleash.NewClient(
		// unleash.WithAppName(config.GetToggleServiceAppName()),
		unleash.WithUrl(config.GetToggleServiceAPIURL()),
		unleash.WithStrategies(&RolloutByGroupIDStrategy{}),
	)
	if err != nil {
		return nil, err
	}
	return &ToggleClient{unleashClient: *client}, nil

}

// Close closes the underlying Unleash client
func (c *ToggleClient) Close() error {
	return c.unleashClient.Close()
}

// GetEnabledFeatures returns the names of enabled features for the given user
func (c *ToggleClient) GetEnabledFeatures(user *account.User) []string {
	return c.unleashClient.GetEnabledFeatures(&unleashcontext.Context{
		Properties: map[string]string{
			"groupID": user.ContextInformation["featureGroupId"].(string),
		},
	})
}

const (
	RolloutByGroupID string = "groupID"
)

// RolloutByGroupIDStrategy the strategy to roll out a feature if the user belongs to a given group
type RolloutByGroupIDStrategy struct {
}

// Name the name of the stragegy. Must match the name on the Unleash server.
func (s *RolloutByGroupIDStrategy) Name() string {
	return "rolloutByGroupId"
}

// IsEnabled returns `true` if the given context is compatible with the settings configured on the Unleash server
func (s *RolloutByGroupIDStrategy) IsEnabled(settings map[string]interface{}, ctx *unleashcontext.Context) bool {
	fmt.Printf("Checking %v vs %v", spew.Sdump(settings), spew.Sdump(ctx))
	return settings[RolloutByGroupID] == ctx.Properties[RolloutByGroupID]
}
