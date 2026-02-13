package contracts

// CloudQueryRole represents the role of a CloudQuery plugin.
type CloudQueryRole string

const (
	// CloudQueryRoleSource indicates a source plugin that reads data from external systems.
	CloudQueryRoleSource CloudQueryRole = "source"
	// CloudQueryRoleDestination indicates a destination plugin that writes data to external systems.
	// NOTE: Destination plugins are reserved but not yet supported.
	CloudQueryRoleDestination CloudQueryRole = "destination"
)

// IsValid returns true if the role is a recognized CloudQuery role.
func (r CloudQueryRole) IsValid() bool {
	switch r {
	case CloudQueryRoleSource, CloudQueryRoleDestination:
		return true
	}
	return false
}

// IsSupported returns true if the role is currently implemented.
func (r CloudQueryRole) IsSupported() bool {
	return r == CloudQueryRoleSource
}

// CloudQuerySpec defines CloudQuery-specific configuration within a DataPackage manifest.
type CloudQuerySpec struct {
	// Role is the plugin role: "source" or "destination".
	Role CloudQueryRole `yaml:"role" json:"role"`
	// Tables is the list of table names this plugin provides.
	Tables []string `yaml:"tables,omitempty" json:"tables,omitempty"`
	// GRPCPort is the port the gRPC server listens on. Default: 7777.
	GRPCPort int `yaml:"grpcPort,omitempty" json:"grpcPort,omitempty"`
	// Concurrency is the max number of concurrent table resolvers. Default: 10000.
	Concurrency int `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
}

// Default fills in default values for optional fields.
func (s *CloudQuerySpec) Default() {
	if s.GRPCPort == 0 {
		s.GRPCPort = 7777
	}
	if s.Concurrency == 0 {
		s.Concurrency = 10000
	}
}
