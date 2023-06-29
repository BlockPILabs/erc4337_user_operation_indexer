package version

var (
	Version string
)

func VersionFull() string {
	return "indexer-" + Version
}
