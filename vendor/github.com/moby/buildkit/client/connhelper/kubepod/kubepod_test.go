package kubepod

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpecFromURL(t *testing.T) {
	cases := map[string]*Spec{
		"kube-pod://podname": {
			Pod: "podname",
		},
		"kube-pod://podname?container=containername&namespace=nsname&context=ctxname": {
			Context: "ctxname", Namespace: "nsname", Pod: "podname", Container: "containername",
		},
		"kube-pod://":                     nil,
		"kube-pod://unsupported_pod_name": nil,
	}
	for s, expected := range cases {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		got, err := SpecFromURL(u)
		if expected != nil {
			require.NoError(t, err)
			require.EqualValues(t, expected, got, s)
		} else {
			require.Error(t, err, s)
		}
	}
}
