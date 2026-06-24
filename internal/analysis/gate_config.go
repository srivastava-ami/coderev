package analysis

type GateConfig struct {
	Blockers   int `toml:"max_blockers"`
	Majors     int `toml:"max_majors"`
	Advisories int `toml:"max_advisories"`
	Total      int `toml:"max_total"`
}
