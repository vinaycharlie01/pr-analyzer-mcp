package entity

import "time"

type AnalysisResult struct {
	ID              string
	PullRequest     *PullRequest
	ExecutiveSummary string
	BusinessPurpose  string
	TechnicalPurpose string
	FilesChanged     []FileAnalysis
	Dependencies     []Dependency
	MigrationPlan    *MigrationPlan
	MigrationChecklist []ChecklistItem
	ArchitectureImpact *ArchitectureImpact
	DatabaseImpact     *DatabaseImpact
	ConfigurationImpact *ConfigurationImpact
	KubernetesImpact   *KubernetesImpact
	ValidationSteps    []ValidationStep
	RollbackStrategy   *RollbackStrategy
	CreatedAt          time.Time
}

type FileAnalysis struct {
	Path            string
	Purpose         string
	ChangeType      FileStatus
	Impact          string
	RelatedFiles    []string
	Dependencies    []string
}

type Dependency struct {
	Name        string
	Version     string
	Type        DependencyType
	Required    bool
	Description string
	Source      string
	Target      string
}

type DependencyType string

const (
	DependencyTypePackage   DependencyType = "package"
	DependencyTypeService   DependencyType = "service"
	DependencyTypeDatabase  DependencyType = "database"
	DependencyTypeAPI       DependencyType = "api"
	DependencyTypeInternal  DependencyType = "internal"
)

type MigrationPlan struct {
	Title       string
	Description string
	Steps       []MigrationStep
	Risks       []Risk
	Effort      EffortLevel
	Timeline    string
}

type MigrationStep struct {
	Order       int
	Title       string
	Description string
	Commands    []string
	Validation  string
	Rollback    string
}

type Risk struct {
	Level       RiskLevel
	Description string
	Mitigation  string
}

type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
)

type EffortLevel string

const (
	EffortLevelSmall  EffortLevel = "small"
	EffortLevelMedium EffortLevel = "medium"
	EffortLevelLarge  EffortLevel = "large"
)

type ChecklistItem struct {
	ID          string
	Title       string
	Description string
	Required    bool
	Category    string
	Completed   bool
}

type ArchitectureImpact struct {
	LayersAffected []string
	PatternsUsed   []string
	NewComponents  []string
	ModifiedComponents []string
	Description    string
}

type DatabaseImpact struct {
	HasMigrations bool
	Migrations    []DatabaseMigration
	TablesAffected []string
	Description   string
}

type DatabaseMigration struct {
	Name      string
	Type      string
	SQL       string
	Reversible bool
}

type ConfigurationImpact struct {
	EnvVarsAdded    []string
	EnvVarsModified []string
	ConfigFilesChanged []string
	Description     string
}

type KubernetesImpact struct {
	ManifestsChanged []string
	ResourcesAffected []string
	Description      string
}

type ValidationStep struct {
	Order       int
	Title       string
	Description string
	Command     string
	Expected    string
}

type RollbackStrategy struct {
	Description string
	Steps       []string
	Commands    []string
}
