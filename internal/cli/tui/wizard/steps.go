package wizard

// stepID identifies each wizard step.
type stepID int

const (
	stepDetectAgents stepID = iota
	stepSelectAgent
	stepInputName
	stepSelectModel
	stepInputRepos
	stepCompanion
	stepFeatureBranches
	stepResearchPaths
	stepInstall
	stepAgentsMDConfirm
	stepAgentsMDRun
	stepDaemon
	stepDone
)

// stepStatus tracks the state of each step.
type stepStatus int

const (
	statusPending stepStatus = iota
	statusActive
	statusDone
	statusSkipped
	statusError
)

// stepDef holds metadata for a step to render in the sidebar.
type stepDef struct {
	id    stepID
	label string
}

// allSteps is the full ordered list of wizard steps shown in the sidebar.
// Not all are always shown — conditional ones are filtered at render time.
var allSteps = []stepDef{
	{stepDetectAgents, "Detect agents"},
	{stepSelectAgent, "Select agent"},
	{stepInputName, "Project name"},
	{stepSelectModel, "Select model"},
	{stepInputRepos, "Repositories"},
	{stepCompanion, "Companion"},
	{stepFeatureBranches, "Feature branches"},
	{stepResearchPaths, "Research paths"},
	{stepInstall, "Install"},
	{stepAgentsMDConfirm, "Generate docs"},
	{stepAgentsMDRun, "Analyzing repos"},
	{stepDaemon, "Start daemon"},
}
