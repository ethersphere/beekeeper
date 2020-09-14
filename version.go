package beekeeper

var (
	version = "0.2.21" // manually set semantic version number
	commit  string     // automatically set git commit hash

	// Version TODO
	Version = func() string {
		if commit != "" {
			return version + "-" + commit
		}
		return version + "-dev"
	}()
)
