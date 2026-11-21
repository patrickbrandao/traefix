package mesh

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// EntryPointSummary is what an instance tells the others of one of its entry points, so that they can tell
// when the same name hides another configuration: the middlewares of an entry point only run where a request
// arrives, and a router of the mesh uses the entry point of the same name on every instance.
type EntryPointSummary struct {
	Protocol    string   `json:"protocol"`
	Fingerprint bool     `json:"fingerprint,omitempty"`
	Middlewares []string `json:"middlewares,omitempty"`
}

// SetEntryPointSummaries records what this instance tells the others of its entry points.
func (m *Mesh) SetEntryPointSummaries(summaries map[string]EntryPointSummary) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entryPointSummaries = summaries
}

// compareConfiguration compares the entry points another instance told of with the ones of this instance,
// and warns when the ones with the same name differ.
// An entry point that only one of the instances has is not a difference: the instances need not have the same ones.
func (m *Mesh) compareConfiguration(theirs hello) {
	if theirs.EntryPoints == nil {
		// An instance of a version that does not tell its entry points.
		return
	}

	m.mu.Lock()
	ours := m.entryPointSummaries
	m.mu.Unlock()

	differences := compareEntryPoints(ours, theirs.EntryPoints)

	m.mu.Lock()
	m.mismatches[theirs.NodeID] = differences
	m.mu.Unlock()

	m.setConfigMismatch(theirs.NodeID, len(differences))

	if len(differences) == 0 {
		return
	}

	m.warnOnce("mismatch/"+theirs.NodeID+"/"+strings.Join(differences, "\x00"), func() {
		m.logger.Warn().Str("peer", theirs.NodeID).Strs("differences", differences).
			Msg("An instance of the mesh has entry points with the same names and another configuration: a request is not handled the same way whichever instance it reaches")
	})
}

// compareEntryPoints returns the differences between the entry points of the same name, in the order of their names.
func compareEntryPoints(ours, theirs map[string]EntryPointSummary) []string {
	var differences []string

	for _, name := range slices.Sorted(maps.Keys(ours)) {
		other, ok := theirs[name]
		if !ok {
			continue
		}

		own := ours[name]

		if own.Protocol != other.Protocol {
			differences = append(differences, fmt.Sprintf("%s: protocol %s here, %s there", name, own.Protocol, other.Protocol))
		}

		if own.Fingerprint != other.Fingerprint {
			differences = append(differences, fmt.Sprintf("%s: fingerprint %s here, %s there", name, onOff(own.Fingerprint), onOff(other.Fingerprint)))
		}

		if !slices.Equal(own.Middlewares, other.Middlewares) {
			differences = append(differences, fmt.Sprintf("%s: middlewares %s here, %s there", name, listOf(own.Middlewares), listOf(other.Middlewares)))
		}
	}

	return differences
}

// mismatchOf returns the differences of configuration with another instance, as last compared.
func (m *Mesh) mismatchOf(nodeID string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.mismatches[nodeID]
}

func onOff(enabled bool) string {
	if enabled {
		return "on"
	}

	return "off"
}

func listOf(names []string) string {
	return "[" + strings.Join(names, ", ") + "]"
}
