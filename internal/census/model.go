package census

type Policy struct {
	Schema        string       `json:"schema"`
	ID            string       `json:"id"`
	Precedence    []string     `json:"precedence"`
	UnknownFields []string     `json:"unknown_fields"`
	Cells         []PolicyCell `json:"cells"`
}

type PolicyCell struct {
	ID        string `json:"id"`
	Proof     string `json:"proof"`
	Indicator string `json:"indicator"`
	Activity  string `json:"activity"`
}

type Manifest struct {
	Schema           string       `json:"schema"`
	ScenarioID       string       `json:"scenario_id"`
	ExpectedDecision string       `json:"expected_decision"`
	Freshness        string       `json:"freshness"`
	Authority        Authority    `json:"authority"`
	Obligations      []Obligation `json:"obligations"`
}

type Authority struct {
	RepositoryWrites        int `json:"repository_writes"`
	LocalTestExecutions     int `json:"local_test_executions"`
	InfrastructureMutations int `json:"infrastructure_mutations"`
	ProviderInstallAttempts int `json:"provider_install_attempts"`
	NetworkMutationAttempts int `json:"network_mutation_attempts"`
}

type Obligation struct {
	ID                 string `json:"id"`
	SourcePath         string `json:"source_path"`
	IRPath             string `json:"ir_path"`
	GeneratedPath      string `json:"generated_path"`
	ImplementationKind string `json:"implementation_kind"`
}

type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
	ObligationID  string   `json:"obligation_id,omitempty"`
}

type Refutation struct {
	Stage          string `json:"stage"`
	Step           string `json:"step"`
	Reason         string `json:"reason"`
	ObligationID   string `json:"obligation_id,omitempty"`
	Counterexample string `json:"counterexample"`
}

type CellResult struct {
	ID        string `json:"id"`
	Activity  string `json:"activity"`
	Proof     string `json:"proof"`
	Indicator string `json:"indicator"`
	State     string `json:"state"`
	Reason    string `json:"reason,omitempty"`
}

type Metrics struct {
	ObligationsTotal        int `json:"obligations_total"`
	GeneratedBound          int `json:"generated_bound"`
	HandwrittenGo           int `json:"handwritten_go"`
	UnknownObligations      int `json:"unknown_obligations"`
	RefutedObligations      int `json:"refuted_obligations"`
	SourceFilesObserved     int `json:"source_files_observed"`
	IRFilesObserved         int `json:"ir_files_observed"`
	GeneratedFilesObserved  int `json:"generated_files_observed"`
	SemanticRelations       int `json:"semantic_relations"`
	ReplayComparisons       int `json:"replay_comparisons"`
	ReplayMismatches        int `json:"replay_mismatches"`
	RepositoryWrites        int `json:"repository_writes"`
	LocalTestExecutions     int `json:"local_test_executions"`
	InfrastructureMutations int `json:"infrastructure_mutations"`
	ProviderInstallAttempts int `json:"provider_install_attempts"`
	NetworkMutationAttempts int `json:"network_mutation_attempts"`
}

type Report struct {
	Schema           string            `json:"schema"`
	ScenarioID       string            `json:"scenario_id"`
	ExpectedDecision string            `json:"expected_decision"`
	Decision         string            `json:"decision"`
	PolicyID         string            `json:"policy_id"`
	Score            string            `json:"score"`
	Cells            []CellResult      `json:"cells"`
	Unknowns         []Unknown         `json:"unknowns"`
	Refutations      []Refutation      `json:"refutations"`
	NextFrontier     []string          `json:"next_frontier"`
	FileDigests      map[string]string `json:"file_digests"`
	Metrics          Metrics           `json:"metrics"`
}
