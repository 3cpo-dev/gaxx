package core

import "context"

// Orchestrator is the entrypoint for coordinating fleets and tasks.
type Orchestrator struct{}

func NewOrchestrator() *Orchestrator { return &Orchestrator{} }

func (o *Orchestrator) Health(ctx context.Context) error { return ctx.Err() }
