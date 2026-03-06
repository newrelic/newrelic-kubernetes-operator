package info

// Version of this library
var (
	Name    = "newrelic-kubernetes-operator"
	Version = "dev"
)

const RepoURL = "https://github.com/newrelic/newrelic-kubernetes-operator"

func UserAgent() string {
	return Name + "/" + Version + " (" + RepoURL + ")"
}
