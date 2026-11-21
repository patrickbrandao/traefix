package dynamic

// +k8s:deepcopy-gen=true

// RouterMesh sets how the mesh between the Traefix instances of a cluster treats a router (Traefix).
type RouterMesh struct {
	// Priority chooses between the instances that have the router. The default, 0, prefers the instance that received
	// the request. Any other value sends the requests to the instances with the highest value.
	Priority int `json:"priority,omitempty" toml:"priority,omitempty" yaml:"priority,omitempty" export:"true"`
	// Export tells whether the router is announced to the other instances.
	// When unset, the cluster mesh export option decides.
	Export *bool `json:"export,omitempty" toml:"export,omitempty" yaml:"export,omitempty" export:"true"`
}
