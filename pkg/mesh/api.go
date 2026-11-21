package mesh

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"
)

// apiStatus is the answer of the API of the mesh.
type apiStatus struct {
	NodeID    string   `json:"nodeID"`
	Role      string   `json:"role"`
	Advertise []string `json:"advertise"`
	// Links are the links this instance opens to the other instances.
	Links []apiLink `json:"links"`
	// Inbound are the instances that opened a link to this one.
	Inbound []apiInbound `json:"inbound"`
	// Exported are the routers this instance announces.
	Exported []Route `json:"exported"`
	// Imported are the routes the other instances announce, and what this instance made of them.
	Imported []ImportedRoute `json:"imported"`
}

type apiLink struct {
	NodeID      string     `json:"nodeID,omitempty"`
	Role        string     `json:"role,omitempty"`
	Address     string     `json:"address,omitempty"`
	Addresses   []string   `json:"addresses"`
	Sources     []string   `json:"sources"`
	Up          bool       `json:"up"`
	Transport   string     `json:"transport,omitempty"`
	Since       time.Time  `json:"since"`
	RTTSeconds  float64    `json:"rttSeconds,omitempty"`
	Error       string     `json:"error,omitempty"`
	LastContact *time.Time `json:"lastContact,omitempty"`
	// Routes is how many routers the instance announces to this one.
	Routes int `json:"routes"`
	// ConfigMismatch are the entry points that the instance has with the same name and another configuration.
	ConfigMismatch []string `json:"configMismatch,omitempty"`
}

type apiInbound struct {
	NodeID    string    `json:"nodeID"`
	Role      string    `json:"role"`
	Version   string    `json:"version,omitempty"`
	Remote    string    `json:"remote"`
	Advertise []string  `json:"advertise"`
	LastSeen  time.Time `json:"lastSeen"`
}

// APIHandler returns the handler of the API of the mesh: the links of this instance, the routers it announces,
// and what it made of the routes of the other instances.
func (m *Mesh) APIHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		status := m.Status()

		answer := apiStatus{
			NodeID:    status.NodeID,
			Role:      status.Role,
			Advertise: status.Advertise,
			Links:     []apiLink{},
			Inbound:   []apiInbound{},
		}

		m.mu.Lock()
		answer.Exported = slices.Clone(m.exported)
		answer.Imported = slices.Clone(m.importReport)

		routes := map[string]int{}
		for nodeID, table := range m.imported {
			routes[nodeID] = len(table.routes)
		}
		m.mu.Unlock()

		// A satellite has no address to tell.
		if answer.Advertise == nil {
			answer.Advertise = []string{}
		}

		if answer.Imported == nil {
			answer.Imported = []ImportedRoute{}
		}

		for _, link := range status.Links {
			entry := apiLink{
				NodeID:     link.NodeID,
				Role:       link.Role,
				Address:    link.Address,
				Addresses:  link.Addresses,
				Sources:    link.Sources,
				Up:         link.Up,
				Transport:  link.Transport,
				Since:      link.Since,
				RTTSeconds: link.RTT.Seconds(),
				Error:      link.Error,
				Routes:     routes[link.NodeID],
			}

			if link.NodeID != "" {
				entry.ConfigMismatch = m.mismatchOf(link.NodeID)
			}

			if !link.LastContact.IsZero() {
				entry.LastContact = &link.LastContact
			}

			answer.Links = append(answer.Links, entry)
		}

		for _, inbound := range status.Inbound {
			answer.Inbound = append(answer.Inbound, apiInbound(inbound))
		}

		rw.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(rw).Encode(answer); err != nil {
			m.logger.Debug().Err(err).Msg("Unable to answer a request to the API of the mesh")
		}
	})
}
