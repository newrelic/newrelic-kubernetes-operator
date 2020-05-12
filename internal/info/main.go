package info

// Version of this library
var (
	Name    string = "newrelic-kubernetes-operator"
	Version string = "dev"
)

const RepoURL = "https://github.com/newrelic/newrelic-kubernetes-operator"

func UserAgent() string {
	return Name + "/" + Version + " (" + RepoURL + ")"
}
