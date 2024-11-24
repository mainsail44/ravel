package state

import (
	"context"
	"fmt"

	"github.com/valyentdev/ravel/api"
	"github.com/valyentdev/ravel/core/cluster"
	"github.com/valyentdev/ravel/ravel/db"
)

type State struct {
	clusterState cluster.ClusterState
	db           *db.DB
}

func New(clusterState cluster.ClusterState, db *db.DB) *State {
	return &State{
		clusterState: clusterState,
		db:           db,
	}
}

func (s *State) CreateMachine(machine cluster.Machine, mv api.MachineVersion) error {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err = tx.CreateMachine(ctx, machine); err != nil {
		return fmt.Errorf("failed to create machine on pg: %w", err)
	}

	if err = tx.CreateMachineVersion(ctx, mv); err != nil {
		return fmt.Errorf("failed to create machine version on pg: %w", err)
	}

	err = s.clusterState.CreateMachine(ctx, machine, mv)
	if err != nil {
		return fmt.Errorf("failed to create machine on corro: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (s *State) UpdateMachine(machine cluster.Machine) error {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if err = tx.UpdateMachine(ctx, machine); err != nil {
		return fmt.Errorf("failed to update machine on pg: %w", err)
	}

	if err = s.clusterState.UpdateMachine(ctx, machine); err != nil {
		return fmt.Errorf("failed to update machine on corro: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (s *State) DestroyMachine(ctx context.Context, id string) error {
	err := s.db.DestroyMachine(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to destroy machine on pg: %w", err)
	}

	err = s.clusterState.DestroyMachine(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to destroy machine on corro: %w", err)
	}

	return nil
}

func (s *State) CreateGateway(gateway api.Gateway) error {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if err = tx.CreateGateway(ctx, gateway); err != nil {
		return fmt.Errorf("failed to create gateway on pg: %w", err)
	}

	err = s.clusterState.CreateGateway(ctx, gateway)
	if err != nil {
		return fmt.Errorf("failed to create gateway on corro: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (s *State) DeleteGateway(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err = tx.DeleteGateway(ctx, id); err != nil {
		return fmt.Errorf("failed to delete gateway on pg: %w", err)
	}

	err = s.clusterState.DeleteGateway(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete gateway on corro: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}