package main

import (
	"encoding/json"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"net/http"
	"time"
)

type PatchOP string

const (
	OP_ADD     PatchOP = "add"
	OP_REPLACE PatchOP = "replace"
	OP_REMOVE  PatchOP = "remove"
)

type Patch struct {
	OP    PatchOP     `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

func serveMutatePods(w http.ResponseWriter, r *http.Request) {
	serve(w, r, mutatePods)
}

func mutatePods(ar v1.AdmissionReview) *v1.AdmissionResponse {
	klog.V(2).Info("mutating pods")

	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		klog.Errorf("expect resource to be %s", podResource)
		return nil
	}

	raw := ar.Request.Object.Raw
	pod := corev1.Pod{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &pod); err != nil {
		klog.Error(err)
		return toAdmissionResponse(err)
	}

	reviewResponse := v1.AdmissionResponse{}
	reviewResponse.Allowed = true

	if matching(ar, pod) {
		klog.V(2).Info("matched pods")
		if marshal, err := json.Marshal(generatePatch(ar, pod)); err != nil {
			klog.Error(err)
			return toAdmissionResponse(err)
		} else {
			reviewResponse.Patch = marshal
		}

		pt := v1.PatchTypeJSONPatch
		reviewResponse.PatchType = &pt
	} else {
		klog.Warning("not match %s", podResource)
	}

	return &reviewResponse
}

// Match Rule,NameSpaces And Label use Kubernetes, Other use this.
func matching(ar v1.AdmissionReview, pod corev1.Pod) bool {
	if config.TriggerENV {
		if len(pod.Spec.Containers) > 0 {
			for _, container := range pod.Spec.Containers {
				if len(container.Env) > 0 {
					for _, env := range container.Env {
						if env.Name == "SWKAC_ENABLE" {
							if env.Value == "true" {
								return true
							}
						}
					}
				}
			}
		}
	}
	return !config.TriggerENV
}

func containerMatching(container corev1.Container) bool {
	if config.TriggerENV {
		if len(container.Env) > 0 {
			for _, env := range container.Env {
				if env.Name == "SWKAC_ENABLE" {
					if env.Value == "true" {
						return true
					}
				}
			}
		}
	}
	return !config.TriggerENV
}

func generatePatch(ar v1.AdmissionReview, pod corev1.Pod) []Patch {
	var patches []Patch

	// addLabels
	if len(pod.Labels) == 0 {
		patches = append(patches, Patch{OP: OP_ADD, Path: "/metadata/labels", Value: make(map[string]string)})
	}
	patches = append(patches, Patch{OP: OP_ADD, Path: "/metadata/labels/skywalking-enabled", Value: "enabled"})
	patches = append(patches, Patch{OP: OP_ADD, Path: "/metadata/labels/skywalking-timestamp", Value: time.Now()})
	//
	//// addVolume
	//swVolumeQuantity, _ := resource.ParseQuantity("200Mi")
	//swVolume := corev1.Volume{
	//	Name: "skywalking",
	//	VolumeSource: corev1.VolumeSource{
	//		EmptyDir: &corev1.EmptyDirVolumeSource{
	//			Medium:    corev1.StorageMediumDefault,
	//			SizeLimit: &swVolumeQuantity,
	//		},
	//	},
	//}
	//if len(pod.Spec.Volumes) == 0 {
	//	patches = append(patches, Patch{OP: OP_ADD, Path: "/spec/volumes", Value: [0]struct{}{}})
	//}
	//patches = append(patches, Patch{OP: OP_ADD, Path: "/spec/volumes/", Value: swVolume})
	//
	//// addInitContainer
	//initContainer := corev1.Container{
	//	Name:  "skywalking",
	//	Image: config.SWImage,
	//	VolumeMounts: []corev1.VolumeMount{
	//		{
	//			Name:      "skywalking",
	//			MountPath: "/opt/skywalking",
	//		},
	//	},
	//}
	//if len(pod.Spec.InitContainers) == 0 {
	//	patches = append(patches, Patch{OP: OP_ADD, Path: "/spec/initContainers", Value: [0]struct{}{}})
	//}
	//patches = append(patches, Patch{OP: OP_ADD, Path: "/spec/initContainers/", Value: initContainer})
	//
	//// container cycle
	//for ic, container := range pod.Spec.Containers {
	//	if containerMatching(container) {
	//		mountPath := fmt.Sprintf("/spec/containers/%d/volumeMounts", ic)
	//
	//		// volumeMount
	//		mount := corev1.VolumeMount{
	//			Name:      "skywalking",
	//			MountPath: "/opt/skywalking",
	//		}
	//		if len(container.VolumeMounts) == 0 {
	//			patches = append(patches, Patch{OP: OP_ADD, Path: mountPath, Value: [0]struct{}{}})
	//		}
	//		patches = append(patches, Patch{OP: OP_ADD, Path: mountPath + "/", Value: mount})
	//
	//		// add Init Command and Pods Env
	//		envPath := fmt.Sprintf("/spec/containers/%d/env", ic)
	//		envSWArg := "-javaagent:/opt/skywalking/skywalking-agent.jar"
	//		envOP := OP_ADD
	//		if len(container.Env) != 0 {
	//			for ie, env := range container.Env {
	//				if env.Name == "JAVA_TOOL_OPTIONS" {
	//					if len(env.Value) != 0 {
	//						envSWArg = env.Value + " " + envSWArg
	//						envPath = envPath + "/" + string(ie)
	//						envOP = OP_REPLACE
	//					}
	//				}
	//			}
	//		}
	//		if len(container.Env) == 0 {
	//			patches = append(patches, Patch{OP: OP_ADD, Path: envPath, Value: [0]struct{}{}})
	//		}
	//		if envOP == OP_REPLACE {
	//			patches = append(patches, Patch{OP: envOP, Path: envPath, Value: envSWArg})
	//		} else {
	//			patches = append(patches, Patch{OP: envOP, Path: envPath + "/",
	//				Value: corev1.EnvVar{Name: "JAVA_TOOL_OPTIONS", Value: envSWArg}})
	//		}
	//	}
	//}
	//if klog.V(2) {
	//	res, _ := json.Marshal(patches)
	//	klog.V(2).Info("Patchs: ", string(res))
	//}
	return patches
}
