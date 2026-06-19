package inbound

import (
	"context"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
)

// PRAnalyzerPort defines the inbound port for PR analysis use cases.
type PRAnalyzerPort interface {
	AnalyzePR(ctx context.Context, req AnalyzePRRequest) (*entity.AnalysisResult, error)
	ExplainChange(ctx context.Context, req ExplainChangeRequest) (*ChangeExplanation, error)
	GenerateMigrationPlan(ctx context.Context, req MigrationPlanRequest) (*entity.MigrationPlan, error)
	AnalyzeDependencies(ctx context.Context, req DependencyRequest) ([]entity.Dependency, error)
	GenerateArchitectureMap(ctx context.Context, req ArchitectureRequest) (*entity.ArchitectureImpact, error)
	FindRelatedFiles(ctx context.Context, req RelatedFilesRequest) ([]string, error)
	FindRequiredDependencies(ctx context.Context, req RequiredDepsRequest) ([]entity.Dependency, error)
	CompareRepositories(ctx context.Context, req CompareReposRequest) (*ComparisonResult, error)
	GenerateMigrationChecklist(ctx context.Context, req ChecklistRequest) ([]entity.ChecklistItem, error)
	ExplainCodeFlow(ctx context.Context, req CodeFlowRequest) (*CodeFlowExplanation, error)
	GenerateFeatureSummary(ctx context.Context, req FeatureSummaryRequest) (*FeatureSummary, error)
	GenerateMigrationDocumentation(ctx context.Context, req MigrationDocRequest) (*MigrationDocumentation, error)
}

type AnalyzePRRequest struct {
	Platform   entity.PlatformType
	Repository valueobject.RepositoryRef
	PRNumber   valueobject.PRNumber
}

type ExplainChangeRequest struct {
	Platform   entity.PlatformType
	Repository valueobject.RepositoryRef
	PRNumber   valueobject.PRNumber
	FilePath   string
}

type ChangeExplanation struct {
	FilePath    string
	Why         string
	What        string
	How         string
	Impact      string
	Risks       []entity.Risk
}

type MigrationPlanRequest struct {
	SourcePlatform entity.PlatformType
	SourceRepo     valueobject.RepositoryRef
	PRNumber       valueobject.PRNumber
	TargetRepo     valueobject.RepositoryRef
}

type DependencyRequest struct {
	Platform   entity.PlatformType
	Repository valueobject.RepositoryRef
	PRNumber   valueobject.PRNumber
}

type ArchitectureRequest struct {
	Platform   entity.PlatformType
	Repository valueobject.RepositoryRef
	PRNumber   valueobject.PRNumber
}

type RelatedFilesRequest struct {
	Platform   entity.PlatformType
	Repository valueobject.RepositoryRef
	PRNumber   valueobject.PRNumber
}

type RequiredDepsRequest struct {
	Platform   entity.PlatformType
	Repository valueobject.RepositoryRef
	PRNumber   valueobject.PRNumber
}

type CompareReposRequest struct {
	SourcePlatform entity.PlatformType
	SourceRepo     valueobject.RepositoryRef
	TargetPlatform entity.PlatformType
	TargetRepo     valueobject.RepositoryRef
}

type ComparisonResult struct {
	Source       entity.Repository
	Target       entity.Repository
	CommonFiles  []string
	Differences  []FileDifference
	Suggestions  []string
}

type FileDifference struct {
	Path        string
	Description string
}

type ChecklistRequest struct {
	Platform   entity.PlatformType
	Repository valueobject.RepositoryRef
	PRNumber   valueobject.PRNumber
}

type CodeFlowRequest struct {
	Platform   entity.PlatformType
	Repository valueobject.RepositoryRef
	PRNumber   valueobject.PRNumber
	EntryPoint string
}

type CodeFlowExplanation struct {
	EntryPoint  string
	Flow        []FlowStep
	Description string
}

type FlowStep struct {
	Order       int
	Function    string
	File        string
	Description string
}

type FeatureSummaryRequest struct {
	Platform   entity.PlatformType
	Repository valueobject.RepositoryRef
	PRNumber   valueobject.PRNumber
}

type FeatureSummary struct {
	Title           string
	BusinessValue   string
	TechnicalDetail string
	AffectedAreas   []string
}

type MigrationDocRequest struct {
	Platform   entity.PlatformType
	Repository valueobject.RepositoryRef
	PRNumber   valueobject.PRNumber
	TargetRepo valueobject.RepositoryRef
}

type MigrationDocumentation struct {
	Title             string
	ExecutiveSummary  string
	TechnicalOverview string
	MigrationSteps    []entity.MigrationStep
	Checklist         []entity.ChecklistItem
	RollbackPlan      *entity.RollbackStrategy
	ValidationSteps   []entity.ValidationStep
}
