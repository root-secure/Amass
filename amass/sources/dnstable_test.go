package sources

import (
	"testing"

	"github.com/root-secure/Amass/amass/core"
)

func TestDNSTable(t *testing.T) {
	if *networkTest == false {
		return
	}

	config := setupConfig(domainTest)
	bus, out := setupEventBus(core.NewNameTopic)
	defer bus.Stop()

	srv := NewDNSTable(config, bus)

	result := testService(srv, out)
	if result < expectedTest {
		t.Errorf("Found %d names, expected at least %d instead", result, expectedTest)
	}
}
