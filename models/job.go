package models

import "time"

// Job represents a job/project in the LMS system
type Job struct {
	ProjectNumber    string
	ProjectName      string
	EngineerInitials string
	DateAssigned     time.Time
	DueDate          time.Time
}

// FormatDateAssigned returns the assigned date in MM/DD/YYYY format
func (j *Job) FormatDateAssigned() string {
	return j.DateAssigned.Format("01/02/2006")
}

// FormatDueDate returns the due date in MM/DD/YYYY format
func (j *Job) FormatDueDate() string {
	return j.DueDate.Format("01/02/2006")
}
