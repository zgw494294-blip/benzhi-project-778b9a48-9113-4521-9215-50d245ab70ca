package domain

import "time"

type CaseStatus string

const (
	StatusDraft       CaseStatus = "draft"
	StatusRemediating CaseStatus = "remediating"
	StatusReview      CaseStatus = "review"
	StatusApproved    CaseStatus = "approved"
	StatusFrozen      CaseStatus = "frozen"
	StatusPermitted   CaseStatus = "permitted"
)

type ArtifactCase struct {
	CaseID             string     `json:"caseId"`
	AccessionCode      string     `json:"accessionCode"`
	Title              string     `json:"title"`
	MaterialProfile    string     `json:"materialProfile"`
	DyeSensitivity     string     `json:"dyeSensitivity"`
	FragileAreas       string     `json:"fragileAreas"`
	HistoricalLuxHours float64    `json:"historicalLuxHours"`
	AnnualLuxHourLimit float64    `json:"annualLuxHourLimit"`
	Status             CaseStatus `json:"status"`
	Version            int64      `json:"version"`
	CreatedBy          string     `json:"createdBy"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	MeasurementStale   bool       `json:"measurementStale"`
}

type LightingSlot struct {
	Name    string  `json:"name"`
	Lux     float64 `json:"lux"`
	Minutes int     `json:"minutes"`
}

type DisplayPlanRevision struct {
	RevisionID           string         `json:"revisionId"`
	CaseID               string         `json:"caseId"`
	RevisionNumber       int            `json:"revisionNumber"`
	CabinetCode          string         `json:"cabinetCode"`
	LightingSlots        []LightingSlot `json:"lightingSlots"`
	DailyOpenMinutes     int            `json:"dailyOpenMinutes"`
	DisplayStartDate     string         `json:"displayStartDate"`
	DisplayEndDate       string         `json:"displayEndDate"`
	RestRotationDays     int            `json:"restRotationDays"`
	UVProtection         bool           `json:"uvProtection"`
	SupersedesRevisionID string         `json:"supersedesRevisionId,omitempty"`
	SubmittedBy          string         `json:"submittedBy"`
	SubmittedAt          time.Time      `json:"submittedAt"`
}

type CalculationItem struct {
	Label   string  `json:"label"`
	Formula string  `json:"formula"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
}
type ExposureAssessment struct {
	AssessmentID            string            `json:"assessmentId"`
	CaseID                  string            `json:"caseId"`
	RevisionID              string            `json:"revisionId"`
	RuleSetVersion          string            `json:"ruleSetVersion"`
	ProjectedLuxHours       float64           `json:"projectedLuxHours"`
	PeakLux                 float64           `json:"peakLux"`
	AnnualRemainingLuxHours float64           `json:"annualRemainingLuxHours"`
	SafetyMarginPercent     float64           `json:"safetyMarginPercent"`
	CalculationBreakdown    []CalculationItem `json:"calculationBreakdown"`
	CalculatedAt            time.Time         `json:"calculatedAt"`
	AnnualBreakdown         []AnnualExposure  `json:"annualBreakdown,omitempty"`
}

type AnnualExposure struct {
	Year                int     `json:"year"`
	LightingDays        int     `json:"lightingDays"`
	ProjectedLuxHours   float64 `json:"projectedLuxHours"`
	HistoricalLuxHours  float64 `json:"historicalLuxHours"`
	RemainingLuxHours   float64 `json:"remainingLuxHours"`
	SafetyMarginPercent float64 `json:"safetyMarginPercent"`
}

type RiskFinding struct {
	FindingID           string     `json:"findingId"`
	CaseID              string     `json:"caseId"`
	AssessmentID        string     `json:"assessmentId"`
	RuleCode            string     `json:"ruleCode"`
	Severity            string     `json:"severity"`
	Summary             string     `json:"summary"`
	EvidenceRequirement string     `json:"evidenceRequirement"`
	Status              string     `json:"status"`
	ResolutionNote      string     `json:"resolutionNote,omitempty"`
	Evidence            string     `json:"evidence,omitempty"`
	ResolvedBy          string     `json:"resolvedBy,omitempty"`
	ResolvedAt          *time.Time `json:"resolvedAt,omitempty"`
}

type ReviewDecision struct {
	DecisionID, CaseID, RevisionID, ReviewerID, Outcome, Reason string
	DecidedAt                                                   time.Time
	CaseVersion                                                 int64
}
type DisplayPermit struct {
	PermitID         string    `json:"permitId"`
	CaseID           string    `json:"caseId"`
	FrozenRevisionID string    `json:"frozenRevisionId"`
	EvidenceDigest   string    `json:"evidenceDigest"`
	Conditions       []string  `json:"conditions"`
	ValidFrom        time.Time `json:"validFrom"`
	ValidUntil       time.Time `json:"validUntil"`
	IssuedBy         string    `json:"issuedBy"`
	IssuedAt         time.Time `json:"issuedAt"`
	VerificationCode string    `json:"verificationCode"`
}

type AuditEvent struct {
	Sequence                         int64 `json:"sequence"`
	CaseID, EventType, Actor, Detail string
	OccurredAt                       time.Time
}
type EvidenceView struct {
	Case         ArtifactCase          `json:"case"`
	Revisions    []DisplayPlanRevision `json:"revisions"`
	Assessments  []ExposureAssessment  `json:"assessments"`
	Findings     []RiskFinding         `json:"findings"`
	Decisions    []ReviewDecision      `json:"decisions"`
	Permit       *DisplayPermit        `json:"permit,omitempty"`
	Audit        []AuditEvent          `json:"audit"`
	FrozenDigest string                `json:"frozenDigest,omitempty"`
	Responses    []ReviewResponse      `json:"responses,omitempty"`
}

type ReviewResponse struct {
	ResponseID  string    `json:"responseId"`
	CaseID      string    `json:"caseId"`
	DecisionID  string    `json:"decisionId"`
	RevisionID  string    `json:"revisionId"`
	Explanation string    `json:"explanation"`
	Actor       string    `json:"actor"`
	CreatedAt   time.Time `json:"createdAt"`
}
