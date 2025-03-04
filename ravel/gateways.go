package ravel

import (
	"context"
	"math/rand"

	"github.com/valyentdev/ravel/api"
	"github.com/valyentdev/ravel/internal/id"
)

type Gateway = api.Gateway

func (r *Ravel) GetGateway(ctx context.Context, namespace string, fleet string, gateway string) (Gateway, error) {
	f, err := r.GetFleet(ctx, namespace, fleet)
	if err != nil {
		return Gateway{}, err
	}

	return r.state.GetGateway(ctx, f.Namespace, f.Id, gateway)
}

func (r *Ravel) ListGateways(ctx context.Context, namespace string, fleet string) ([]Gateway, error) {
	f, err := r.GetFleet(ctx, namespace, fleet)
	if err != nil {
		return nil, err
	}

	return r.state.ListGatewaysOnFleet(ctx, namespace, f.Id)

}

type CreateGatewayOptions = api.CreateGatewayPayload

func (r *Ravel) CreateGateway(ctx context.Context, namespace string, fleet string, options CreateGatewayOptions) (Gateway, error) {
	if options.Name != "" {
		if err := validateObjectName(options.Name); err != nil {
			return Gateway{}, err
		}
	}
	f, err := r.GetFleet(ctx, namespace, fleet)
	if err != nil {
		return Gateway{}, err
	}

	var name string
	if options.Name == "" {
		name = generateGatewayName(f.Name)
	} else {
		name = options.Name
	}

	gateway := Gateway{
		Id:         id.GeneratePrefixed("gw"),
		Name:       name,
		Namespace:  namespace,
		Protocol:   "https",
		FleetId:    f.Id,
		TargetPort: options.TargetPort,
	}

	err = r.state.CreateGateway(gateway)
	if err != nil {
		return Gateway{}, err
	}

	return gateway, nil
}

func (r *Ravel) DeleteGateway(ctx context.Context, namespace string, fleet string, gateway string) error {
	g, err := r.GetGateway(ctx, namespace, fleet, gateway)
	if err != nil {
		return err
	}

	return r.state.DeleteGateway(ctx, g.Id)
}

var gwNameAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func generateSuffix(length int) string {
	var suffix string
	for i := 0; i < length; i++ {
		suffix += string(gwNameAlphabet[rand.Intn(len(gwNameAlphabet))])
	}

	return suffix
}

// If fleet name is "my-fleet", the generated gateway name will be something like "my-fleet-f3a9"
func generateGatewayName(fleet string) string {
	base := fleet
	if len(fleet) > 58 {
		base = fleet[58:]

		// Remove any trailing hyphens
		for base[len(base)-1] == '-' {
			base = base[:len(base)-1]
		}
	}
	return base + "-" + generateSuffix(4)
}
