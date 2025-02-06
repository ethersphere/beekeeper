package beekeeper

var (
	version string // automatically set semantic version number
	commit  string // automatically set git commit hash

	// Version TODO
	Version = func() string {
		if commit != "" {
			return version + "-" + commit
		}
		return version + "-dev"
	}()
)
