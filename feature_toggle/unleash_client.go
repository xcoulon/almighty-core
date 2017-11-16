package feature_toggle

import (
	"context"

	unleash "github.com/Unleash/unleash-client-go"
	"github.com/fabric8-services/fabric8-wit/account"
)

// ToggleClient the toggle client
type ToggleClient struct {
	unleashClient unleash.Client
}

// ToggleServiceConfiguration the configuration to the Toggle service
type ToggleServiceConfiguration interface {
	GetToggleServiceAPIURL() string
}

// NewFeatureToggleClient returns a new client to the toggle feature service
func NewFeatureToggleClient(ctx context.Context, config ToggleServiceConfiguration) (*ToggleClient, error) {
	client, err := unleash.NewClient(
		unleash.WithAppName("fabric8"),
		unleash.WithUrl("http://localhost:4242/api/"),
		unleash.WithStrategies(&RolloutByGroupIDStrategy{}),
	)
	if err != nil {
		return nil, err
	}
	return &ToggleClient{unleashClient: *client}, nil

}

// GetEnabledFeatures returns the list of enabled features for the given user
func (c *ToggleClient) GetEnabledFeatures(user account.User) ([]string, error) {
	// c.unleashClient.
	return nil, nil
}

type RolloutByGroupIDStrategy struct {
}

func (s *RolloutByGroupIDStrategy) Name() string {
	return "rolloutByGroupId"
}

func (s *RolloutByGroupIDStrategy) IsEnabled(map[string]interface{}, *context.Context) bool {
	return true
}
