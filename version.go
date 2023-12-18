package beekeeper

var (
	version = "0.15.0" // manually set semantic version number
	commit  string     // automatically set git commit hash

	// Version TODO
	Version = func() string {
		if commit != "" {
			return version + "-" + commit
		}
		return version + "-dev"
	}()
)
