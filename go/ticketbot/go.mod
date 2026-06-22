module github.com/TicketsBot-cloud/ticketbot

go 1.25.0

// Dependencies (worker, dashboard, and the shared modules) are resolved from local
// source via the workspace ../../go.work. Versions here are placeholders the workspace
// overrides; they are finalized in the unification step.
require (
	github.com/TicketsBot-cloud/common v0.0.0-00010101000000-000000000000
	github.com/TicketsBot-cloud/dashboard v0.0.0-00010101000000-000000000000
	github.com/TicketsBot-cloud/worker v0.0.0-00010101000000-000000000000
)
