package commands

import (
	"agent-memory/cmd/agent/harness"
)

var registeredHarness *harness.AgentHarness

func Register(h *harness.AgentHarness) {
	registeredHarness = h
}

func GetHarness() *harness.AgentHarness {
	return registeredHarness
}
