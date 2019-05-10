package v1

const TypeUrl = "networking.internal.knative.dev/v1alpha1/ClusterIngress"

type Status_State int32

const (
	// Pending status indicates the resource has not yet been validated
	Status_Pending Status_State = 0
	// Accepted indicates the resource has been validated
	Status_Accepted Status_State = 1
	// Rejected indicates an invalid configuration by the user
	// Rejected resources may be propagated to the xDS server depending on their severity
	Status_Rejected Status_State = 2
)

type Any struct {
	TypeUrl string
	Value   []byte
}

type Status struct {
	State               Status_State
	Reason              string
	ReportedBy          string
	SubresourceStatuses map[string]*Status
}

type Metadata struct {
	Name            string
	Namespace       string
	ResourceVersion string
	Labels          map[string]string
	Annotations     map[string]string
}

type ClusterIngress struct {
	Metadata             Metadata
	Status               Status
	ClusterIngressSpec   *Any
	ClusterIngressStatus *Any
}

var Status_State_name = map[int32]string{
	0: "Pending",
	1: "Accepted",
	2: "Rejected",
}

var Status_State_value = map[string]int32{
	"Pending":  0,
	"Accepted": 1,
	"Rejected": 2,
}

type ClusterIngressList []*ClusterIngress

type TranslatorSnapshots struct {
	Clusteringresses ClusterIngressList
}
