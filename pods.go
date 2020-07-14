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
	"strings"
)

type PatchOP string

const (
	OP_ADD            PatchOP = "add"
	OP_REPLACE        PatchOP = "replace"
	OP_REMOVE         PatchOP = "remove"
	DEFINE_AGENT_PATH         = "/opt/skywalking"
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

	patches = addLabels(ar, pod, patches)

	patches = addSharedVolume(ar, pod, patches)

	patches = addInitContainer(ar, pod, patches)

	// container cycle
	for ic, container := range pod.Spec.Containers {
		if containerMatching(container) {
			patches = addContainerVolumeMount(ar, pod, ic, container, patches)

			patches = addContainerStartAgentCommand(ar, pod, ic, container, patches)

			patches = addContainerCollectorDefine(ar, pod, ic, container, patches)

			patches = addContainerAgentName(ar, pod, ic, container, patches)
		}
	}
	// Debug Patch
	if klog.V(2) {
		res, _ := json.Marshal(patches)
		klog.V(2).Info("Patch: ", string(res))
	}
	return patches
}

// SW_AGENT_NAME
func addContainerAgentName(ar v1.AdmissionReview, pod corev1.Pod, ic int, container corev1.Container, patches []Patch) []Patch {
	envPath := fmt.Sprintf("/spec/containers/%d/env", ic)
	appName := "pod-" + pod.Name

	envCache := map[string]string{}
	if len(container.Env) != 0 {
		for _, env := range container.Env {
			// Already exists, skip it
			if env.Name == "SW_AGENT_NAME" {
				return patches
			}
			envCache[env.Name] = env.Value
		}
	} else {
		patches = append(patches, Patch{OP: OP_ADD, Path: envPath, Value: [0]struct{}{}})
	}

	if host, ok := envCache["HOST"]; ok {
		appName = host
	} else if len(pod.Labels) > 0 {
		// Deployment or Replication Set
		hash, ok := pod.Labels["pod-template-hash"]
		if ok {
			hashIndex := strings.Index(pod.Name, hash)
			appName = pod.Name[0 : hashIndex-1]
		}
	}

	patches = append(patches, Patch{OP: OP_ADD, Path: envPath + "/-", Value: corev1.EnvVar{Name: "SW_AGENT_NAME", Value: appName}})

	return patches
}

// SW_AGENT_COLLECTOR_BACKEND_SERVICES
func addContainerCollectorDefine(ar v1.AdmissionReview, pod corev1.Pod, ic int, container corev1.Container, patches []Patch) []Patch {
	envPath := fmt.Sprintf("/spec/containers/%d/env", ic)
	if len(container.Env) != 0 {
		for _, env := range container.Env {
			// Already exists, skip it
			if env.Name == "SW_AGENT_COLLECTOR_BACKEND_SERVICES" {
				return patches
			}
		}
	} else {
		patches = append(patches, Patch{OP: OP_ADD, Path: envPath, Value: [0]struct{}{}})
	}
	patches = append(patches, Patch{OP: OP_ADD, Path: envPath + "/-",
		Value: corev1.EnvVar{Name: "SW_AGENT_COLLECTOR_BACKEND_SERVICES", Value: config.SWAgentCollectorBackendServices}})
	return patches
}

func addContainerStartAgentCommand(ar v1.AdmissionReview, pod corev1.Pod, ic int, container corev1.Container, patches []Patch) []Patch {
	envPath := fmt.Sprintf("/spec/containers/%d/env", ic)
	envSWArg := "-javaagent:" + DEFINE_AGENT_PATH + "/skywalking-agent.jar"
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
	if len(container.Env) == 0 {
		patches = append(patches, Patch{OP: OP_ADD, Path: envPath, Value: [0]struct{}{}})
	}

	if envOP == OP_REPLACE {
		patches = append(patches, Patch{OP: envOP, Path: envPath, Value: envSWArg})
	} else {
		patches = append(patches, Patch{OP: envOP, Path: envPath + "/-",
			Value: corev1.EnvVar{Name: "JAVA_TOOL_OPTIONS", Value: envSWArg}})
	}
	return patches
}

// ic: index of container
func addContainerVolumeMount(ar v1.AdmissionReview, pod corev1.Pod, ic int, container corev1.Container, patches []Patch) []Patch {
	mountPath := fmt.Sprintf("/spec/containers/%d/volumeMounts", ic)

	mount := corev1.VolumeMount{
		Name:      volumeName(string(ar.Request.UID)),
		MountPath: DEFINE_AGENT_PATH,
	}

	if len(container.VolumeMounts) == 0 {
		patches = append(patches, Patch{OP: OP_ADD, Path: mountPath, Value: [0]struct{}{}})
	}
	patches = append(patches, Patch{OP: OP_ADD, Path: mountPath + "/-", Value: mount})
	return patches
}

func addInitContainer(ar v1.AdmissionReview, pod corev1.Pod, patches []Patch) []Patch {
	initContainer := corev1.Container{
		Name:  initContainerName(string(ar.Request.UID)),
		Image: config.SWImage,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volumeName(string(ar.Request.UID)),
				MountPath: DEFINE_AGENT_PATH,
			},
		},
		Env: []corev1.EnvVar{{
			Name:  "AGENT_HOME",
			Value: DEFINE_AGENT_PATH,
		}},
	}

	if len(pod.Spec.InitContainers) == 0 {
		patches = append(patches, Patch{OP: OP_ADD, Path: "/spec/initContainers", Value: [0]struct{}{}})
	}
	patches = append(patches, Patch{OP: OP_ADD, Path: "/spec/initContainers/-", Value: initContainer})
	return patches
}

// Skywalking Agent Volume
func addSharedVolume(ar v1.AdmissionReview, pod corev1.Pod, patches []Patch) []Patch {
	swVolumeQuantity, _ := resource.ParseQuantity("200Mi")
	swVolume := corev1.Volume{
		Name: volumeName(string(ar.Request.UID)),
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium:    corev1.StorageMediumDefault,
				SizeLimit: &swVolumeQuantity,
			},
		},
	}

	if len(pod.Spec.Volumes) == 0 {
		patches = append(patches, Patch{OP: OP_ADD, Path: "/spec/volumes", Value: [0]struct{}{}})
	}
	patches = append(patches, Patch{OP: OP_ADD, Path: "/spec/volumes/-", Value: swVolume})
	return patches
}

// addLabels
func addLabels(ar v1.AdmissionReview, pod corev1.Pod, patches []Patch) []Patch {
	if len(pod.Labels) == 0 {
		patches = append(patches, Patch{OP: OP_ADD, Path: "/metadata/labels", Value: make(map[string]string)})
	}
	patches = append(patches, Patch{OP: OP_ADD, Path: "/metadata/labels/skywalking", Value: "enabled"})
	patches = append(patches, Patch{OP: OP_ADD, Path: "/metadata/labels/skywalking-volume",
		Value: volumeName(string(ar.Request.UID))})
	return patches
}

func volumeName(id string) string {
	if len(id) > 8 {
		id = id[0:8]
	}
	return "sw-volume-" + id
}

func initContainerName(id string) string {
	if len(id) > 8 {
		id = id[0:8]
	}
	return "sw-init-" + id
}
