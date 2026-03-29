package baseline

import (
	"encoding/json"
	"os"
	"time"
)

type Entry struct {
	Repo      string `json:"repo"`
	RuleID    string `json:"rule_id"`
	Status    string `json:"status"`
	FirstSeen string `json:"first_seen"`
}

type Baseline struct {
	GeneratedAt string  `json:"generated_at"`
	Entries     []Entry `json:"entries"`
}

func Load(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func Save(path string, b *Baseline) error {
	b.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
