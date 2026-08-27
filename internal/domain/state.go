package domain

func (c *ArtifactCase) EnsureMutable() error {
	if c.Status == StatusFrozen || c.Status == StatusPermitted {
		return ErrForbidden
	}
	return nil
}

func (c *ArtifactCase) Move(next CaseStatus) error {
	allowed := map[CaseStatus]map[CaseStatus]bool{
		StatusDraft:       {StatusRemediating: true, StatusReview: true},
		StatusRemediating: {StatusRemediating: true, StatusReview: true},
		StatusReview:      {StatusApproved: true, StatusRemediating: true},
		StatusApproved:    {StatusFrozen: true},
		StatusFrozen:      {StatusPermitted: true},
	}
	if !allowed[c.Status][next] {
		return ErrForbidden
	}
	c.Status = next
	return nil
}

func CheckReviewer(applicant, reviewer string) error {
	if applicant == "" || reviewer == "" {
		return &RuleError{"reviewerId", "申请人和复核人不能为空"}
	}
	if applicant == reviewer {
		return ErrSeparation
	}
	return nil
}
