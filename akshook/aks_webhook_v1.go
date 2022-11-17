package akshook

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/wI2L/jsondiff"
	"io/ioutil"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
	"net/http"
)

var (
	runtimeScheme  = runtime.NewScheme()
	codeFactory    = serializer.NewCodecFactory(runtimeScheme)
	deserializer   = codeFactory.UniversalDeserializer()
	logPatchedYAML = true
)

var (
	INIT_COMMAND  = []string{"/bin/sh", "-c", "source /app/init-appinsights.sh; cp /app/* /config/"}
	INIT_VOLMOUNT = []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      "appinsights-config",
			MountPath: "/config/",
		},
	}
)

const (
	INSIGHT_CONNSTR = "appinsights.connstr"
	INSIGHT_ROLE    = "appinsights.role"

	INIT_NAME  = "copy"
	INIT_IMAGE = "nikawang.azurecr.io/spring/app-insights-agent:v1"
)

type AksWebhookParam struct {
	Port     int
	CertFile string
	KeyFile  string
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type WebhookServer struct {
	Server              *http.Server
	WhiteListRegistries []string
}

func (s *WebhookServer) Handler(writer http.ResponseWriter, request *http.Request) {
	var body []byte
	if request.Body != nil {
		if data, err := ioutil.ReadAll(request.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		klog.Error("empty data body")
		http.Error(writer, "empty data body", http.StatusBadRequest)
		return
	}

	// Validate content-type
	contentType := request.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Errorf("Content-Type is %s, but expect application/json", contentType)
		http.Error(writer, "Content-Type invalid, expect application/json", http.StatusBadRequest)
		return
	}

	// Decode Admission Review
	var admissionResponse *admissionv1.AdmissionResponse
	requestedAdmissionReview := admissionv1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		klog.Errorf("Can't decode body: %v", err)
		admissionResponse = &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		}
	} else {
		if request.URL.Path == "/mutate" {
			admissionResponse = s.mutateJsonDiff(&requestedAdmissionReview)
		}
	}

	responseAdmissionReview := admissionv1.AdmissionReview{}
	responseAdmissionReview.APIVersion = requestedAdmissionReview.APIVersion
	responseAdmissionReview.Kind = requestedAdmissionReview.Kind
	if admissionResponse != nil {
		responseAdmissionReview.Response = admissionResponse
		if requestedAdmissionReview.Request != nil { // 返回相同的 UID
			responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		}

	}

	klog.Infof("sending response: %v", responseAdmissionReview.Response)
	// send response
	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Errorf("Can't encode response: %v", err)
		http.Error(writer, fmt.Sprintf("Can't encode response: %v", err), http.StatusBadRequest)
		return
	}
	klog.Info("Ready to write response...")

	if _, err := writer.Write(respBytes); err != nil {
		klog.Errorf("Can't write response: %v", err)
		http.Error(writer, fmt.Sprintf("Can't write reponse: %v", err), http.StatusBadRequest)
	}
}

func (s *WebhookServer) mutatePods(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {

	klog.Infof("MutatePods AdmissionReview for Kind=%s, Namespace=%s Name=%s UID=%s",
		ar.Request.Kind.Kind, ar.Request.Namespace, ar.Request.Name, ar.Request.UID)

	pod := &corev1.PodTemplateSpec{}
	if err := json.Unmarshal(ar.Request.Object.Raw, pod); err != nil {
		klog.Errorf("request unmarshal error: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		}
	}
	klog.Infof("Pods Found: %+v", pod)

	annotationMap := pod.ObjectMeta.GetAnnotations()
	klog.Infof("Annotation Found: %s", annotationMap)
	if !mutationRequired(annotationMap) {
		klog.Info("No need to Mutate")
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	klog.Infof("Mutating YAML ...")
	INIT_ENV := []corev1.EnvVar{
		corev1.EnvVar{
			Name:  "CONNECTION_STRING",
			Value: annotationMap[INSIGHT_CONNSTR],
		},
		corev1.EnvVar{
			Name:  "ROLE_NAME",
			Value: annotationMap[INSIGHT_ROLE],
		},
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
		Name:         INIT_NAME,
		Image:        INIT_IMAGE,
		Command:      INIT_COMMAND,
		Env:          INIT_ENV,
		VolumeMounts: INIT_VOLMOUNT,
	})

	klog.Infof("Add initContainer: %+v", pod.Spec.InitContainers)

	initContainerBytes, err := json.Marshal(&pod.Spec.InitContainers)
	if err != nil {
		klog.Errorf("Init Container unmarshal error: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		}
	}

	patch := []patchOperation{
		patchOperation{
			Op:    "add",
			Path:  "/spec/template/spec",
			Value: initContainerBytes,
		},
	}
	patchBytes, err := json.Marshal(&patch)
	if err != nil {
		klog.Errorf("patch marshal error: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		}
	}

	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}
}

func (s *WebhookServer) mutateJsonDiff(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	// Deployment、Service -> annotations： AnnotationMutateKey， AnnotationStatusKey
	req := ar.Request

	var (
		deployment appsv1.Deployment
	)

	klog.Infof("AdmissionReview for Kind=%s, Namespace=%s Name=%s UID=%s",
		req.Kind.Kind, req.Namespace, req.Name, req.UID)

	switch req.Kind.Kind {
	case "Deployment":
		if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
			klog.Errorf("Can't not unmarshal raw object: %v", err)
			return &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				},
			}

		}
	case "Service":
		klog.Errorf("No need to Mutate Service")
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	default:
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Can't handle the kind(%s) object", req.Kind.Kind),
			},
		}
	}

	if !mutationRequired(deployment.ObjectMeta.GetAnnotations()) {
		klog.Info("No need to Mutate")
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	newDeploy := deployment.DeepCopy()
	newPodSpec := mutateContainers(&newDeploy.Spec.Template.Spec, deployment.ObjectMeta.GetAnnotations())

	if logPatchedYAML {
		klog.Info("\n---------begin mumated yaml---------")
		bytes, err := json.Marshal(newPodSpec)
		if err == nil {
			yamlStr, err := yaml.JSONToYAML(bytes)
			if err == nil {
				klog.Info("\n" + string(yamlStr))
			}
		}
		klog.Info("\n---------ended mumated yaml---------")
	}

	patch, err := jsondiff.Compare(deployment, newDeploy)
	if err != nil {
		klog.Errorf("json diff marshal error: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		}
	}

	patchBytes, err := json.MarshalIndent(patch, "", "    ")
	if err != nil {
		klog.Errorf("patch marshal error: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		}
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func mutationRequired(annotations map[string]string) bool {

	klog.Infof("Inside mutated Required, annotations : %s", annotations)

	if _, ok := annotations[INSIGHT_CONNSTR]; ok {
		return true
	}

	if _, ok := annotations[INSIGHT_ROLE]; ok {
		return true
	}

	return false
}

func mutateContainers(deploy *corev1.PodSpec, annotations map[string]string) (result *corev1.PodSpec) {
	INIT_ENV := []corev1.EnvVar{
		corev1.EnvVar{
			Name:  "CONNECTION_STRING",
			Value: annotations[INSIGHT_CONNSTR],
		},
		corev1.EnvVar{
			Name:  "ROLE_NAME",
			Value: annotations[INSIGHT_ROLE],
		},
	}

	if len(deploy.InitContainers) == 0 {
		deploy.InitContainers = []corev1.Container{
			{
				Name:         INIT_NAME,
				Image:        INIT_IMAGE,
				Command:      INIT_COMMAND,
				Env:          INIT_ENV,
				VolumeMounts: INIT_VOLMOUNT,
			},
		}
		klog.Info("\nmutate add initContainer success!")
	}

	deploy.Containers[0].Command = []string{"/bin/sh", "-c", "cp /config/* /app/ ; java -javaagent:applicationinsights-agent-3.3.1.jar -jar department-service-1.2-SNAPSHOT.jar"}

	return deploy
}
