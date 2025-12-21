package registry

// Game represents a game server definition from the registry
type Game struct {
	Name          string            `yaml:"name"`
	DisplayName   string            `yaml:"display_name"`
	Description   string            `yaml:"description"`
	Image         string            `yaml:"image"`
	Ports         Ports             `yaml:"ports"`
	InternalPorts Ports             `yaml:"internal_ports"`
	Protocols     Protocols         `yaml:"protocols"`
	Volumes       []string          `yaml:"volumes"`
	ConfigSchema  map[string]any    `yaml:"config_schema"`
}

// Ports defines port mappings
type Ports struct {
	Player int `yaml:"player"`
	RCON   int `yaml:"rcon"`
}

// Protocols defines protocol types for ports
type Protocols struct {
	Player string `yaml:"player"`
	RCON   string `yaml:"rcon"`
}

// DefaultProtocol returns the protocol or "tcp" if not set
func (p Protocols) DefaultProtocol(port string) string {
	var proto string
	switch port {
	case "player":
		proto = p.Player
	case "rcon":
		proto = p.RCON
	}
	if proto == "" {
		return "tcp"
	}
	return proto
}
