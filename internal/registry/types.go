package registry

// Game represents a game server definition from the registry
type Game struct {
	Name          string    `yaml:"name"`
	DisplayName   string    `yaml:"display_name"`
	Description   string    `yaml:"description"`
	Image         string    `yaml:"image"`
	Ports         Ports     `yaml:"ports"`
	InternalPorts Ports     `yaml:"internal_ports"`
	Protocols     Protocols `yaml:"protocols"`
	Volumes       []string  `yaml:"volumes"`
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

// DefaultProtocol returns the protocol for a port name, defaulting to "tcp" if not set
// Returns "tcp" if port name is invalid
func (p Protocols) DefaultProtocol(port string) string {
	switch port {
	case "player":
		if p.Player != "" {
			return p.Player
		}
	case "rcon":
		if p.RCON != "" {
			return p.RCON
		}
	default:
		// Invalid port name, log but default to tcp
		return "tcp"
	}
	return "tcp"
}
