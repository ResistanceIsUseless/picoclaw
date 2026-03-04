package graph

import "fmt"

// EntityType defines the type of entity in the knowledge graph
type EntityType string

// Web/Network domain entities
const (
	EntityDomain       EntityType = "domain"
	EntitySubdomain    EntityType = "subdomain"
	EntityIP           EntityType = "ip"
	EntityPort         EntityType = "port"
	EntityService      EntityType = "service"
	EntityEndpoint     EntityType = "endpoint"
	EntityParameter    EntityType = "parameter"
	EntityCVE          EntityType = "cve"
	EntityCredential   EntityType = "credential"
	EntityCertificate  EntityType = "certificate"
	EntityTechnology   EntityType = "technology"
)

// Source code domain entities
const (
	EntityFunction      EntityType = "function"
	EntityStruct        EntityType = "struct"
	EntityVariable      EntityType = "variable"
	EntityAllocation    EntityType = "allocation"
	EntityTrustBoundary EntityType = "trust_boundary"
	EntitySink          EntityType = "sink"
	EntitySource        EntityType = "source"
)

// Binary/Firmware domain entities
const (
	EntityBinary         EntityType = "binary"
	EntitySharedLibrary  EntityType = "shared_library"
	EntityFirmwareImage  EntityType = "firmware_image"
	EntityFileSystem     EntityType = "filesystem"
)

// RelationType defines the type of relationship between entities
type RelationType string

const (
	// Web/Network relationships
	RelationSubdomainOf    RelationType = "subdomain_of"    // subdomain -> domain
	RelationResolvesTo     RelationType = "resolves_to"     // domain -> IP
	RelationHostsService   RelationType = "hosts_service"   // IP -> service
	RelationExposesPort    RelationType = "exposes_port"    // IP -> port
	RelationRunsOn         RelationType = "runs_on"         // service -> port
	RelationHasEndpoint    RelationType = "has_endpoint"    // service -> endpoint
	RelationAcceptsParam   RelationType = "accepts_param"   // endpoint -> parameter
	RelationVulnerableTo   RelationType = "vulnerable_to"   // service/endpoint -> CVE
	RelationUsesTech       RelationType = "uses_tech"       // service -> technology
	RelationHasCert        RelationType = "has_cert"        // service -> certificate
	RelationAuthenticates  RelationType = "authenticates"   // credential -> service

	// Source code relationships
	RelationCalls          RelationType = "calls"           // function -> function
	RelationFlowsTo        RelationType = "flows_to"        // source -> sink (data flow)
	RelationAllocates      RelationType = "allocates"       // function -> allocation
	RelationFrees          RelationType = "frees"           // function -> allocation
	RelationCrossesBoundary RelationType = "crosses_boundary" // data -> trust_boundary
	RelationContains       RelationType = "contains"        // struct -> variable

	// Binary/Firmware relationships
	RelationLinks          RelationType = "links"           // binary -> shared_library
	RelationPackages       RelationType = "packages"        // firmware -> filesystem
	RelationIncludes       RelationType = "includes"        // filesystem -> binary
)

// EntityDefinition defines the schema for an entity type
type EntityDefinition struct {
	Type               EntityType   `json:"type"`
	DiscoverableProps  []string     `json:"discoverable_props"` // properties that can be discovered
	RequiredProps      []string     `json:"required_props"`     // properties that must be present
	HighInterestProps  []string     `json:"high_interest_props"` // properties with high exploration priority
	DefaultInterest    float64      `json:"default_interest"`   // base interest score (0.0 - 1.0)
}

// EntityRegistry manages entity type definitions
type EntityRegistry struct {
	definitions map[EntityType]*EntityDefinition
}

// NewEntityRegistry creates a new entity registry with default definitions
func NewEntityRegistry() *EntityRegistry {
	registry := &EntityRegistry{
		definitions: make(map[EntityType]*EntityDefinition),
	}

	// Register default entity types
	registry.registerDefaultEntities()

	return registry
}

// registerDefaultEntities registers all standard entity types
func (r *EntityRegistry) registerDefaultEntities() {
	// Web/Network entities
	r.Register(&EntityDefinition{
		Type: EntitySubdomain,
		DiscoverableProps: []string{
			"ip_addresses",
			"ports",
			"services",
			"technologies",
			"endpoints",
			"certificates",
		},
		RequiredProps:     []string{"name"},
		HighInterestProps: []string{"ports", "services"},
		DefaultInterest:   0.7,
	})

	r.Register(&EntityDefinition{
		Type: EntityIP,
		DiscoverableProps: []string{
			"open_ports",
			"os",
			"services",
			"asn",
			"geolocation",
		},
		RequiredProps:     []string{"address"},
		HighInterestProps: []string{"open_ports", "services"},
		DefaultInterest:   0.8,
	})

	r.Register(&EntityDefinition{
		Type: EntityPort,
		DiscoverableProps: []string{
			"service",
			"version",
			"banner",
			"vulnerabilities",
		},
		RequiredProps:     []string{"number", "protocol"},
		HighInterestProps: []string{"vulnerabilities", "version"},
		DefaultInterest:   0.6,
	})

	r.Register(&EntityDefinition{
		Type: EntityService,
		DiscoverableProps: []string{
			"version",
			"configuration",
			"vulnerabilities",
			"authentication",
			"endpoints",
		},
		RequiredProps:     []string{"name"},
		HighInterestProps: []string{"vulnerabilities", "authentication"},
		DefaultInterest:   0.8,
	})

	r.Register(&EntityDefinition{
		Type: EntityEndpoint,
		DiscoverableProps: []string{
			"parameters",
			"authentication_required",
			"vulnerabilities",
			"http_methods",
			"response_codes",
		},
		RequiredProps:     []string{"url"},
		HighInterestProps: []string{"vulnerabilities", "parameters"},
		DefaultInterest:   0.7,
	})

	r.Register(&EntityDefinition{
		Type: EntityParameter,
		DiscoverableProps: []string{
			"type",
			"validation",
			"injectable",
			"sink_type",
		},
		RequiredProps:     []string{"name"},
		HighInterestProps: []string{"injectable", "sink_type"},
		DefaultInterest:   0.9, // Parameters are high-interest for exploitation
	})

	// Source code entities
	r.Register(&EntityDefinition{
		Type: EntityFunction,
		DiscoverableProps: []string{
			"calls",
			"called_by",
			"allocations",
			"external_input",
			"dangerous_sinks",
			"sanitizers_present",
			"buffer_overflow_path",
			"use_after_free_path",
		},
		RequiredProps:     []string{"name", "file", "line"},
		HighInterestProps: []string{"dangerous_sinks", "external_input", "buffer_overflow_path"},
		DefaultInterest:   0.6,
	})

	r.Register(&EntityDefinition{
		Type: EntityAllocation,
		DiscoverableProps: []string{
			"size",
			"freed",
			"use_after_free_path",
			"double_free_path",
		},
		RequiredProps:     []string{"site"},
		HighInterestProps: []string{"use_after_free_path", "double_free_path"},
		DefaultInterest:   0.7,
	})

	r.Register(&EntityDefinition{
		Type: EntityTrustBoundary,
		DiscoverableProps: []string{
			"sources",
			"flows_to",
			"sanitizers",
		},
		RequiredProps:     []string{"name"},
		HighInterestProps: []string{"flows_to"}, // Where does untrusted data go?
		DefaultInterest:   0.9,
	})

	// Binary/Firmware entities
	r.Register(&EntityDefinition{
		Type: EntitySharedLibrary,
		DiscoverableProps: []string{
			"version",
			"cves",
			"reachable_functions",
		},
		RequiredProps:     []string{"name"},
		HighInterestProps: []string{"cves", "reachable_functions"},
		DefaultInterest:   0.8,
	})

	r.Register(&EntityDefinition{
		Type: EntityFirmwareImage,
		DiscoverableProps: []string{
			"filesystem",
			"kernel_version",
			"mitigations",
			"libraries",
		},
		RequiredProps:     []string{"name"},
		HighInterestProps: []string{"kernel_version", "mitigations"},
		DefaultInterest:   0.7,
	})
}

// Register adds an entity definition to the registry
func (r *EntityRegistry) Register(def *EntityDefinition) {
	r.definitions[def.Type] = def
}

// Get retrieves an entity definition by type
func (r *EntityRegistry) Get(entityType EntityType) (*EntityDefinition, error) {
	def, exists := r.definitions[entityType]
	if !exists {
		return nil, fmt.Errorf("entity type %q not registered", entityType)
	}
	return def, nil
}

// GetDiscoverableProperties returns the list of discoverable properties for an entity type
func (r *EntityRegistry) GetDiscoverableProperties(entityType EntityType) ([]string, error) {
	def, err := r.Get(entityType)
	if err != nil {
		return nil, err
	}
	return def.DiscoverableProps, nil
}

// GetHighInterestProperties returns properties with high exploration priority
func (r *EntityRegistry) GetHighInterestProperties(entityType EntityType) ([]string, error) {
	def, err := r.Get(entityType)
	if err != nil {
		return nil, err
	}
	return def.HighInterestProps, nil
}

// GetDefaultInterest returns the base interest score for an entity type
func (r *EntityRegistry) GetDefaultInterest(entityType EntityType) (float64, error) {
	def, err := r.Get(entityType)
	if err != nil {
		return 0.0, err
	}
	return def.DefaultInterest, nil
}

// IsHighInterestProperty checks if a property is high-interest for an entity type
func (r *EntityRegistry) IsHighInterestProperty(entityType EntityType, propertyName string) bool {
	def, err := r.Get(entityType)
	if err != nil {
		return false
	}

	for _, prop := range def.HighInterestProps {
		if prop == propertyName {
			return true
		}
	}

	return false
}
