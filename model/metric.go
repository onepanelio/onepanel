package model

type Metric struct {
	Name   string
	Value  float64
	Format string `json:"omitempty"`
}
