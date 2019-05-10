package configmap

import (
	"github.com/ledboot/knative-cluster-ingress/pkg/logging"
	"testing"
)

func TestLoadFile(t *testing.T) {
	loggingConfigMap, err := Load("testdata")
	if err != nil {
		t.Errorf("load config fail")
	}
	//t.Log(loggingConfigMap)
	config, err := logging.NewConfigFromMap(loggingConfigMap, "controller")
	if err != nil {
		t.Errorf(err.Error())
	}
	t.Log(config)
}
