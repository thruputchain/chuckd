package k8s

import (
	"fmt"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MakeConfigMap(name string, cfg proto.Config) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string]string{
			"config.json": xdrutil.MustXdrToJsonString(&cfg),
		},
	}
}

func MakeConfigMapsForStatefulSet(serviceName string, cfg []proto.Config) (
	[]*corev1.ConfigMap,
	[]corev1.Volume,
	[]corev1.VolumeMount,
) {

	cs := make([]*corev1.ConfigMap, len(cfg))
	vs := make([]corev1.Volume, len(cfg))
	ms := make([]corev1.VolumeMount, len(cfg))
	for i, cfg := range cfg {
		cfgMapName := fmt.Sprintf("%s-config-%d", serviceName, i+1)
		mountName := fmt.Sprintf("config-%d", i+1)
		cs[i] = MakeConfigMap(cfgMapName, cfg)
		vs[i] = corev1.Volume{
			Name: mountName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cfgMapName,
					},
				},
			},
		}
		ms[i] = corev1.VolumeMount{
			Name:      mountName,
			MountPath: fmt.Sprintf("/configs/config-%d", i+1),
		}
	}
	return cs, vs, ms
}
