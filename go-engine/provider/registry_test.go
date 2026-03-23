package provider_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

func TestRegistryKnownProviders(t *testing.T) {
	reg := provider.DefaultRegistry()
	for _, id := range []string{"anthropic", "openai", "ollama"} {
		p, ok := reg.Get(id)
		if !ok {
			t.Errorf("provider %q không tồn tại trong registry", id)
			continue
		}
		if len(p.Models) == 0 {
			t.Errorf("provider %q phải có ít nhất 1 model", id)
		}
	}
}

func TestRegistryModelCost(t *testing.T) {
	reg := provider.DefaultRegistry()
	p, ok := reg.Get("anthropic")
	if !ok {
		t.Fatal("anthropic phải tồn tại")
	}
	found := false
	for _, m := range p.Models {
		if m.ID == "claude-3-5-sonnet-20241022" {
			found = true
			if m.InputCostPer1K <= 0 || m.OutputCostPer1K <= 0 {
				t.Errorf("model %q phải có cost > 0", m.ID)
			}
		}
	}
	if !found {
		t.Error("claude-3-5-sonnet-20241022 không có trong registry anthropic")
	}
}

func TestRegistryList(t *testing.T) {
	reg := provider.DefaultRegistry()
	list := reg.List()
	if len(list) < 3 {
		t.Errorf("List() phải trả về ít nhất 3 providers, got %d", len(list))
	}
}
