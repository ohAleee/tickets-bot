package main

// Unified TicketsBot entrypoint.
//
// This binary boots, in a single process, every Go subsystem that the upstream
// deployment ran as separate containers:
//   - worker interactions HTTP server  (Discord interaction receipt)
//   - worker gateway RPC consumer       (Redis-stream Discord events)
//   - worker messagequeue listeners     (ticket close / autoclose / close-request timers)
//   - dashboard REST API + websockets
//   - autoclose daemon loop
//   - database view refresher loop
//   - logarchiver HTTP server           (transcript storage)
//
// Implemented in the "Unify Go services into one binary" workstream.
func main() {
	panic("ticketbot: not yet wired — see plan workstream 2")
}
