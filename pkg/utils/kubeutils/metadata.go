package kubeutils

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	res := map[string]string{}
	for k, v := range m {
		res[k] = v
	}
	return res
}
