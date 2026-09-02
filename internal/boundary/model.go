package boundary

const (
	PolicySchema    = "gooo/self-description-boundary/policy/v1"
	InputSchema     = "gooo/self-description-boundary/input/v1"
	ReportSchema    = "gooo/self-description-boundary/report/v1"
	DecisionClosed  = "CLOSED"
	DecisionUnknown = "UNKNOWN"
	DecisionRefuted = "REFUTED"
	FixedPoint      = "FIXED_POINT"
)

var AuthorityStates = []string{
	"GOOO_OWNED",
	"GENERATED_FROM_GOOO",
	"HANDWRITTEN_RUNTIME",
	"BOOTSTRAP_EXTERNAL",
	"UNKNOWN",
	"REFUTED",
}

var OwnedAuthorityStates = map[string]bool{
	"GOOO_OWNED":          true,
	"GENERATED_FROM_GOOO": true,
	"HANDWRITTEN_RUNTIME": true,
	"BOOTSTRAP_EXTERNAL":  true,
}

type Policy struct {
	Schema          string     `json:"schema"`
	ID              string     `json:"id"`
	Release         string     `json:"release"`
	Precedence      []string   `json:"precedence"`
	UnknownFields   []string   `json:"unknown_fields"`
	AuthorityStates []string   `json:"authority_states"`
	FixedPointRule  string     `json:"fixed_point_rule"`
	OutputAuthority string     `json:"output_authority"`
	Cells           []CellSpec `json:"authority_cells"`
}

type CellSpec struct {
	ID            string `json:"id"`
	ExpectedState string `json:"expected_state"`
	EvidenceKind  string `json:"evidence_kind"`
	SemanticRole  string `json:"semantic_role"`
	Activity      string `json:"activity"`
	Proof         string `json:"proof"`
	Indicator     string `json:"indicator"`
	Description   string `json:"description"`
}

type CompiledPolicy struct {
	Schema       string `json:"schema"`
	SourcePath   string `json:"source_path"`
	SourceDigest string `json:"source_digest"`
	Policy       Policy `json:"policy"`
}

type InputManifest struct {
	Schema           string            `json:"schema"`
	ScenarioID       string            `json:"scenario_id"`
	ExpectedDecision string            `json:"expected_decision"`
	Release          ReleaseMetadata   `json:"release"`
	Authority        AuthorityCounters `json:"authority"`
	FixedPoint       *EvidenceRef      `json:"fixed_point"`
	Cells            []CellObservation `json:"cells"`
}

type ReleaseMetadata struct {
	ID             string   `json:"id"`
	SourceRevision string   `json:"source_revision"`
	SourceRelease  string   `json:"source_release"`
	Toolchain      string   `json:"toolchain"`
	InputPaths     []string `json:"input_paths"`
}

type AuthorityCounters struct {
	RepositoryWrites        int `json:"repository_writes"`
	LocalTestExecutions     int `json:"local_test_executions"`
	InfrastructureMutations int `json:"infrastructure_mutations"`
	ProviderInstallAttempts int `json:"provider_install_attempts"`
	NetworkMutationAttempts int `json:"network_mutation_attempts"`
}

type CellObservation struct {
	CellID       string        `json:"cell_id"`
	ClaimedState string        `json:"claimed_state"`
	Method       string        `json:"method"`
	Evidence     []EvidenceRef `json:"evidence"`
}

type EvidenceRef struct {
	Role           string `json:"role"`
	Kind           string `json:"kind"`
	Path           string `json:"path"`
	StartLine      int    `json:"start_line"`
	EndLine        int    `json:"end_line"`
	Marker         string `json:"marker"`
	ExpectedDigest string `json:"expected_digest,omitempty"`
}

type EvidenceRecord struct {
	Role      string   `json:"role"`
	Kind      string   `json:"kind"`
	Path      string   `json:"path"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Digest    *string  `json:"digest"`
	Lines     []string `json:"lines"`
	Status    string   `json:"status"`
}

type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Refutation struct {
	Stage          string `json:"stage"`
	Step           string `json:"step"`
	Reason         string `json:"reason"`
	Counterexample string `json:"counterexample"`
	CellID         string `json:"cell_id,omitempty"`
}

type CellResult struct {
	ID             string           `json:"id"`
	SemanticRole   string           `json:"semantic_role"`
	Description    string           `json:"description"`
	Activity       string           `json:"activity"`
	Proof          string           `json:"proof"`
	Indicator      string           `json:"indicator"`
	AuthorityState string           `json:"authority_state"`
	Evidence       []EvidenceRecord `json:"evidence"`
}

type FixedPointResult struct {
	State    string          `json:"state"`
	Evidence *EvidenceRecord `json:"evidence,omitempty"`
}

type ReplayResult struct {
	Comparisons      int    `json:"comparisons"`
	Mismatches       int    `json:"mismatches"`
	ProjectionDigest string `json:"projection_digest"`
	ReplayDigest     string `json:"replay_digest"`
	State            string `json:"state"`
}

type Report struct {
	Schema           string            `json:"schema"`
	ScenarioID       string            `json:"scenario_id"`
	ExpectedDecision string            `json:"expected_decision"`
	Decision         string            `json:"decision"`
	PolicyID         string            `json:"policy_id"`
	PolicySource     string            `json:"policy_source"`
	PolicyDigest     string            `json:"policy_digest"`
	AuthorityVector  []CellResult      `json:"authority_vector"`
	ProofVector      []string          `json:"proof_vector"`
	IndicatorVector  []string          `json:"indicator_vector"`
	FixedPoint       FixedPointResult  `json:"fixed_point"`
	Unknowns         []Unknown         `json:"unknowns"`
	Refutations      []Refutation      `json:"refutations"`
	NextFrontier     []string          `json:"next_frontier"`
	FileDigests      map[string]string `json:"file_digests"`
	Replay           ReplayResult      `json:"replay"`
	Authority        AuthorityCounters `json:"authority"`
	Release          ReleaseMetadata   `json:"release"`
}
