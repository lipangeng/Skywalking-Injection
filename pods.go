package main

import (
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	if config.triggerENV {
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
	return !config.triggerENV
}

func containerMatching(container corev1.Container) bool {
	if config.triggerENV {
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
	return !config.triggerENV
}

func generatePatch(ar v1.AdmissionReview, pod corev1.Pod) []Patch {
	var patches []Patch

	// addLabels
	patches = append(patches, Patch{OP: OP_ADD, Path: "/metadata/labels",
		Value: struct {
			Status    string    `json:"skywalking"`
			Timestamp time.Time `json:"skywalking-timestamp"`
		}{
			Status:    "enabled",
			Timestamp: time.Now(),
		},
	})

	// addInitContainer
	initContainer := corev1.Container{

	}
	patches = append(patches, Patch{OP: OP_ADD, Path: "/spec/initContainers", Value: initContainer})

	// addVolume
	swVolumeQuantity, _ := resource.ParseQuantity("200Mi")
	swVolume := corev1.Volume{
		Name: "sw",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium:    corev1.StorageMediumDefault,
				SizeLimit: &swVolumeQuantity,
			},
		},
	}
	patches = append(patches, Patch{OP: OP_ADD, Path: "/spec/volumes", Value: swVolume})

	// container cycle
	for ic, container := range pod.Spec.Containers {
		if containerMatching(container) {
			mountPath := fmt.Sprintf("/spec/containers/[%d]/volumeMounts", i)
			// volumeMount
			mount := corev1.VolumeMount{
				Name:      "sw",
				MountPath: "/opt/skywalking",
			}
			patches = append(patches, Patch{OP: OP_ADD, Path: mountPath, Value: mount})

			// add Init Command and Pods Env
			envPath := fmt.Sprintf("/spec/containers/[%d]/env", ic)
			envSWArg := "-javaagent:/opt/skywalking/skywalking-agent.jar"
			envOP := OP_ADD
			if len(container.Env) != 0 {
				for ie, env := range container.Env {
					if env.Name == "JAVA_TOOL_OPTIONS" {
						if len(env.Value) != 0 {
							envSWArg = env.Value + " " + envSWArg
							envPath = envPath + "/" + string(ie)
							envOP = OP_REPLACE
						}
					}
				}
			}
			patches = append(patches, Patch{OP: envOP, Path: envPath, Value: envSWArg})
		}
	}
	return patches
}
