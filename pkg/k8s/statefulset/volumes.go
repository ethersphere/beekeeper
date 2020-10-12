package statefulset

// VolumeConfig ...
type VolumeConfig struct {
	Name        string
	ConfigMap   string
	DefaultMode int32
}

// VolumeConfigFile ...
type VolumeConfigFile struct {
	Name string
}

// VolumeData ...
type VolumeData struct {
	Name string
}

// VolumeLibP2P ...
type VolumeLibP2P struct {
	Name        string
	Secret      string
	DefaultMode int32
	Items       []Item
}

// Item ...
type Item struct {
	Key   string
	Value string
}
