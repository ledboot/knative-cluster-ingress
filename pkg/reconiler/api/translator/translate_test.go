package translator

import (
	"flag"
	"fmt"
	"github.com/hbagdi/go-kong/kong"
	"testing"
)

var (
	baseUrl = flag.String("baseUrl", "", "kong admin baseurl")
)

func TestTranslate(t *testing.T) {
	kongClient, err := kong.NewClient(baseUrl, nil)

	if err != nil {
		fmt.Errorf("build kong client get error :", err)
	}
	root, err := kongClient.Root(nil)
	if err != nil {
		fmt.Errorf("get kong root error:", err)
	}
	fmt.Println(root)
}
