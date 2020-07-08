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
	PatchOP_ADD     PatchOP = "add"
	PatchOP_REPLACE PatchOP = "replace"
	PatchOP_REMOVE  PatchOP = "remove"
)

type Patch struct {
	OP    PatchOP     `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

const ()

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

	if matching(ar) {
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
func matching(ar v1.AdmissionReview) bool {

	return true
}

func generatePatch(ar v1.AdmissionReview, pod corev1.Pod) []Patch {
	var patchs []Patch

	// addLabel,
	patchs = append(patchs, Patch{OP: PatchOP_ADD, Path: "/metadata/labels",
		Value: struct {
			Status    string    `json:"skywalking"`
			Timestamp time.Time `json:"skywalking-timestamp"`
		}{
			Status:    "enabled",
			Timestamp: time.Now(),
		},
	})




	return patchs
}
