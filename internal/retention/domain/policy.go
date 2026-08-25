package domain

import (
	"fmt"
	"time"
)

type Policy struct {
	MetadataTTL   time.Duration
	RawMessageTTL time.Duration
	DeadLetterTTL time.Duration
	BatchSize     int
}

func DefaultPolicy() Policy {
	return Policy{MetadataTTL: 90 * 24 * time.Hour, RawMessageTTL: 30 * 24 * time.Hour, DeadLetterTTL: 180 * 24 * time.Hour, BatchSize: 500}
}

func (p Policy) Validate() error {
	if p.MetadataTTL <= 0 {
		return fmt.Errorf("metadata TTL must be positive")
	}
	if p.RawMessageTTL <= 0 || p.RawMessageTTL > p.MetadataTTL {
		return fmt.Errorf("raw message TTL must be positive and no greater than metadata TTL")
	}
	if p.DeadLetterTTL < p.MetadataTTL {
		return fmt.Errorf("dead-letter TTL must be at least metadata TTL")
	}
	if p.BatchSize < 1 || p.BatchSize > 10000 {
		return fmt.Errorf("batch size must be between 1 and 10000")
	}
	return nil
}
