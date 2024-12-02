package ravel

import (
	"context"
	"crypto/rand"
	"io"
	"time"

	"github.com/oklog/ulid"

	"github.com/valyentdev/ravel/api"
	"github.com/valyentdev/ravel/core/cluster"
	"github.com/valyentdev/ravel/core/config"
	"github.com/valyentdev/ravel/core/errdefs"
	"github.com/valyentdev/ravel/internal/id"
)

func getResources(m config.MachineResourcesTemplates, vcpus int, memory int) (api.Resources, error) {
	for _, c := range m.Combinations {
		if c.VCpus == vcpus {
			for _, mc := range c.MemoryConfigs {
				if mc == memory {
					return api.Resources{
						MemoryMB: mc,
						CpusMHz:  m.VCPUFrequency * vcpus,
					}, nil
				}
			}
		}
	}
	return api.Resources{}, errdefs.NewInvalidArgument("Invalid vcpus and memory config")
}

type CreateMachineOptions struct {
	Region    string            `json:"region"`
	Config    api.MachineConfig `json:"config"`
	SkipStart bool              `json:"skip_start"`
}

func (r *Ravel) CreateMachine(ctx context.Context, namespace string, fleet string, createOptions CreateMachineOptions) (*api.Machine, error) {
	f, err := r.GetFleet(ctx, namespace, fleet)
	if err != nil {
		return nil, err
	}

	ctx = context.Background() // from here we begin to use background context to avoid cancellation of the context passed in and data loss

	versionId := ulid.MustNew(ulid.Now(), rand.Reader).String()
	machine := cluster.Machine{
		Id:             id.Generate(),
		Namespace:      f.Namespace,
		FleetId:        f.Id,
		InstanceId:     id.Generate(),
		MachineVersion: versionId,
		Region:         createOptions.Region,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	cputemplate, ok := r.vcpusTemplates[createOptions.Config.Guest.CpuKind]
	if !ok {
		return nil, errdefs.NewInvalidArgument("Invalid CPU kind")
	}

	resources, err := getResources(cputemplate, createOptions.Config.Guest.Cpus, createOptions.Config.Guest.MemoryMB)
	if err != nil {
		return nil, err
	}

	mv := api.MachineVersion{
		Id:        versionId,
		MachineId: machine.Id,
		Namespace: machine.Namespace,
		Config:    createOptions.Config,
		Resources: resources,
	}

	nodeId, err := r.o.PrepareAllocation(ctx, machine.Region, machine.Id, resources)
	if err != nil {
		return nil, err
	}

	machine.Node = nodeId

	err = r.state.CreateMachine(machine, mv)
	if err != nil {
		return nil, err
	}

	err = r.o.PutMachine(ctx, nodeId, &machine, mv, !createOptions.SkipStart)
	if err != nil {
		return nil, err
	}

	return &api.Machine{
		Id:             machine.Id,
		Namespace:      machine.Namespace,
		FleetId:        machine.FleetId,
		InstanceId:     machine.InstanceId,
		MachineVersion: machine.MachineVersion,
		Region:         machine.Region,
		Config:         createOptions.Config,
		CreatedAt:      machine.CreatedAt,
		UpdatedAt:      machine.UpdatedAt,
		Status:         api.MachineStatusCreated,
	}, nil
}

func (r *Ravel) StartMachine(ctx context.Context, ns, fleet, machineId string) error {
	machine, err := r.getMachine(ctx, ns, fleet, machineId, false)
	if err != nil {
		return err
	}

	return r.o.StartMachineInstance(ctx, machine)
}

func (r *Ravel) StopMachine(ctx context.Context, ns, fleet, machineId string, stopConfig *api.StopConfig) error {
	machine, err := r.getMachine(ctx, ns, fleet, machineId, false)
	if err != nil {
		return err
	}

	return r.o.StopMachineInstance(ctx, machine, stopConfig)
}

func (r *Ravel) ListMachines(ctx context.Context, ns, fleet string, includeDestroyed bool) ([]api.Machine, error) {
	f, err := r.GetFleet(ctx, ns, fleet)
	if err != nil {
		return nil, err
	}

	return r.state.ListAPIMachines(ctx, ns, f.Id, includeDestroyed)
}

func (r *Ravel) DestroyMachine(ctx context.Context, ns, fleet, machineId string, force bool) error {
	m, err := r.getMachine(ctx, ns, fleet, machineId, false)
	if err != nil {
		return err
	}

	return r.o.DestroyMachine(ctx, m, force)
}

func (r *Ravel) getMachine(ctx context.Context, ns, fleet, machineId string, showDestroyed bool) (cluster.Machine, error) {
	f, err := r.GetFleet(ctx, ns, fleet)
	if err != nil {
		return cluster.Machine{}, err
	}

	return r.state.GetMachine(ctx, ns, f.Id, machineId, showDestroyed)
}

func (r *Ravel) GetMachine(ctx context.Context, ns, fleet, machineId string) (*api.Machine, error) {
	f, err := r.GetFleet(ctx, ns, fleet)
	if err != nil {
		return nil, err
	}

	return r.state.GetAPIMachine(ctx, ns, f.Id, machineId)
}

func (r *Ravel) ListMachineVersions(ctx context.Context, ns, fleet, machineId string) ([]api.MachineVersion, error) {
	_, err := r.getMachine(ctx, ns, fleet, machineId, true)
	if err != nil {
		return nil, err
	}
	return r.state.ListMachineVersions(ctx, machineId)
}

func (r *Ravel) GetMachineLogsRaw(ctx context.Context, ns, fleet, machineId string, follow bool) (io.ReadCloser, error) {
	m, err := r.getMachine(ctx, ns, fleet, machineId, false)
	if err != nil {
		return nil, err
	}

	return r.o.GetMachineLogsRaw(ctx, m, follow)
}

func (r *Ravel) ListMachineEvents(ctx context.Context, ns, fleet, machineId string) ([]api.MachineEvent, error) {
	m, err := r.getMachine(ctx, ns, fleet, machineId, true)
	if err != nil {
		return nil, err
	}

	return r.state.ListMachineEvents(ctx, m.Id)
}
