package service

import (
	"fmt"

	"wzap/internal/model"
)

type CapabilityError struct {
	Engine     string
	Capability model.EngineCapability
	Support    model.CapabilitySupport
}

func (e *CapabilityError) Error() string {
	switch e.Support {
	case model.CapabilitySupportPartial:
		return fmt.Sprintf("operation partially supported for %s engine", e.Engine)
	default:
		return fmt.Sprintf("operation not supported for %s engine", e.Engine)
	}
}

func capabilitySupport(engine string, capability model.EngineCapability) model.CapabilitySupport {
	return model.DefaultEngineCapabilityContract.Support(engine, capability)
}

func requireCapability(engine string, capability model.EngineCapability) (model.CapabilitySupport, error) {
	support := capabilitySupport(engine, capability)
	if support == model.CapabilitySupportUnavailable {
		return support, &CapabilityError{
			Engine:     engine,
			Capability: capability,
			Support:    support,
		}
	}

	return support, nil
}
