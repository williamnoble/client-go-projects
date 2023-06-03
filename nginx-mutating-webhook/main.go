package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	v1 "k8s.io/api/admission/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	k8sClientSet *kubernetes.Clientset
)

type ServerConfiguration struct {
	port            int
	certificateFile string
	keyFile         string
}

type patchConfiguration struct {
	Containers []v12.Container `yaml:"Containers"`
	Volumes    []v12.Volume    `yaml:"Volumes"`
}

type patchOperation struct {
	Op    string      `json:"op"`   // Operation
	Path  string      `json:"path"` // Path
	Value interface{} `json:"value,omitempty"`
}

type nginxSidecarConfiguration struct {
	Name            string
	ImageName       string
	ImagePullPolicy v12.PullPolicy
	Port            int
	VolumeMounts    []v12.VolumeMount
}

func generateNginxSideCarConfig(c nginxSidecarConfiguration, volumes []v12.Volume) *patchConfiguration {
	var nginxContainerPort []v12.ContainerPort

	nginxContainer := v12.Container{
		Name:            c.SetContainerNameOrDefault(),
		Image:           c.ImageName,
		ImagePullPolicy: c.ImagePullPolicy,
		Ports: append(nginxContainerPort, v12.ContainerPort{
			ContainerPort: int32(c.Port),
			Protocol:      "TCP",
		}),
		VolumeMounts: c.VolumeMounts,
	}

	sideCars := []v12.Container{nginxContainer}

	return &patchConfiguration{
		Containers: sideCars,
		Volumes:    volumes,
	}
}

func (c nginxSidecarConfiguration) SetContainerNameOrDefault() string {
	containerName := "nginx-webserver"
	if c.Name != "" {
		containerName = c.Name
	}
	return containerName
}

func getPodVolumes(uniqueId string) []v12.Volume {
	var volumes []v12.Volume

	volumes = append(volumes, v12.Volume{
		Name: "nginx-tls-" + uniqueId,
		VolumeSource: v12.VolumeSource{
			Secret: &v12.SecretVolumeSource{SecretName: "sidecar-injector-certs"},
		},
	})

	volumes = append(volumes, v12.Volume{
		Name: "nginx-conf- " + uniqueId,
		VolumeSource: v12.VolumeSource{
			ConfigMap: &v12.ConfigMapVolumeSource{
				LocalObjectReference: v12.LocalObjectReference{Name: "nginx-conf"},
			},
		},
	},
	)
	return volumes
}

func getNginxSideCarConfig(uniqueId string) *patchConfiguration {
	var volumesMount []v12.VolumeMount

	log.Println("generating volume mount count for side car with unique Id ", uniqueId)
	volumesMount = append(volumesMount, v12.VolumeMount{
		Name:      "nginx-conf-" + uniqueId,
		MountPath: "/etc/nginx/nginx.conf",
		SubPath:   "nginx.conf",
	})
	volumesMount = append(volumesMount, v12.VolumeMount{
		Name:      "nginx-tls-" + uniqueId,
		MountPath: "/etc/nginx/ssl",
	})

	return generateNginxSideCarConfig(nginxSidecarConfiguration{
		ImagePullPolicy: v12.PullAlways,
		ImageName:       "nginx:stable",
		Port:            80,
		VolumeMounts:    volumesMount,
	},
		getPodVolumes(uniqueId))
}

func main() {
	var config ServerConfiguration
	flag.IntVar(&config.port, "port", 8443, "Server port for Webhook.")
	flag.StringVar(&config.certificateFile, "certFile", "/etc/webhook/certs/tls.crt", "TLS Certificate")
	flag.StringVar(&config.keyFile, "keyFile", "/etc/webhook/certs/tls.key", "Key Certificate")
	flag.Parse()

	// create ClientSet
	k8sClientSet = createClientSet()

	// setup Routes
	mux := chi.NewRouter()
	mux.Get("/", handleRoot)
	mux.Get("/mutate", handleMutate)

	// sanity test
	podCount()
	err := http.ListenAndServeTLS(":8443", config.certificateFile, config.keyFile, mux)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func addContainer(target, containers []v12.Container, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}

	for _, add := range containers {
		value = add
		path := basePath
		if first {
			first = false
			value = []v12.Container{add}
		} else {
			path = path + "/-"
		}
		fmt.Printf("container json patch Op: %s, Path: %s, Value: %+v", "add", path, value)
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}

	return patch
}

func addVolume(target, volumes []v12.Volume, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}

	for _, add := range volumes {
		value = add
		path := basePath

		if first {
			first = false
			value = []v12.Volume{add}
		} else {
			path = path + "/-"
		}

		log.Printf("volume json patch Op: %s, Path: %s, Value: %+v", "add", path, value)
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}

	return patch
}

func createPatch(pod v12.Pod, sidecarConfig *patchConfiguration) ([]patchOperation, error) {
	fmt.Println("creating json patch of pod for sidecar config")
	var patches []patchOperation
	patches = append(patches, addContainer(pod.Spec.Containers, sidecarConfig.Containers, "/spec/containers")...)
	patches = append(patches, addVolume(pod.Spec.Volumes, sidecarConfig.Volumes, "/spec/volumes")...)

	labels := pod.ObjectMeta.Labels
	labels["nginx-sidecar"] = "applied-from-mutating-webhook"

	patches = append(patches, patchOperation{
		Op:    "add",
		Path:  "/metadata/labels",
		Value: labels,
	})

	return patches, nil

}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("handleRoot"))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	admissionReviewRequest := getAdmissionReviewRequest(w, r)

	var pod v12.Pod
	err := json.Unmarshal(admissionReviewRequest.Request.Object.Raw, &pod)
	if err != nil {
		_ = fmt.Errorf("failed to unmarshal admission requests %s\n", err)
	}
	w.Write([]byte("handleMutate"))
}

func createClientSet() *kubernetes.Clientset {
	k8sConfig := config.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(k8sConfig)

	if err != nil {
		log.Fatal(err)
	}

	return clientSet
}

func podCount() {
	pods, err := k8sClientSet.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to get pods: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Total running pods in cluster: %d", len(pods.Items))
}

func getAdmissionReviewRequest(w http.ResponseWriter, r *http.Request) v1.AdmissionReview {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Failed to read body of admission request ", err)
		os.Exit(1)
	}

	// AdmissionReview is used for both *AdmissionRequest and *AdmissionResponse
	var admissionReviewRequest v1.AdmissionReview

	universalDeserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := universalDeserializer.Decode(body, nil, &admissionReviewRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Failed to deserialize request ", err)
	} else if admissionReviewRequest.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	return admissionReviewRequest
}
