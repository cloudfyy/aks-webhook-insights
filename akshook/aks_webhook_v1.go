package akshook

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"net/http"
)

var (
	runtimeScheme = runtime.NewScheme()
	codeFactory   = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codeFactory.UniversalDeserializer()
)

const (
	INSIGHT_CONNSTR_KEY = "appinsights.connstr"
	INSIGHT_ROLE_KEY    = "appinsights.role"
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
		glog.Error("empty data body")
		http.Error(writer, "empty data body", http.StatusBadRequest)
		return
	}

	// 校验 content-type
	contentType := request.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type is %s, but expect application/json", contentType)
		http.Error(writer, "Content-Type invalid, expect application/json", http.StatusBadRequest)
		return
	}

	// 数据序列化（validate、mutate）请求的数据都是 AdmissionReview
	var admissionResponse *admissionv1.AdmissionResponse
	requestedAdmissionReview := admissionv1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		}
	} else {
		// 序列化成功，也就是说获取到了请求的 AdmissionReview 的数据
		if request.URL.Path == "/mutate" {
			admissionResponse = s.mutate(&requestedAdmissionReview)
		}
	}

	// 构造返回的 AdmissionReview 这个结构体
	responseAdmissionReview := admissionv1.AdmissionReview{}
	// admission/v1
	responseAdmissionReview.APIVersion = requestedAdmissionReview.APIVersion
	responseAdmissionReview.Kind = requestedAdmissionReview.Kind
	if admissionResponse != nil {
		responseAdmissionReview.Response = admissionResponse
		if requestedAdmissionReview.Request != nil { // 返回相同的 UID
			responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		}

	}

	glog.Infof("sending response: %v", responseAdmissionReview.Response)
	// send response
	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(writer, fmt.Sprintf("Can't encode response: %v", err), http.StatusBadRequest)
		return
	}
	glog.Info("Ready to write response...")

	if _, err := writer.Write(respBytes); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(writer, fmt.Sprintf("Can't write reponse: %v", err), http.StatusBadRequest)
	}
}

func (s *WebhookServer) mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	// Deployment、Service -> annotations： AnnotationMutateKey， AnnotationStatusKey
	req := ar.Request

	var (
		objectMeta *metav1.ObjectMeta
	)

	glog.Infof("AdmissionReview for Kind=%s, Namespace=%s Name=%s UID=%s",
		req.Kind.Kind, req.Namespace, req.Name, req.UID)

	switch req.Kind.Kind {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
			glog.Errorf("Can't not unmarshal raw object: %v", err)
			return &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				},
			}

		}
		objectMeta = &deployment.ObjectMeta
	case "Service":
		var service corev1.Service
		if err := json.Unmarshal(req.Object.Raw, &service); err != nil {
			glog.Errorf("Can't not unmarshal raw object: %v", err)
			return &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				},
			}
		}
		objectMeta = &service.ObjectMeta
	default:
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Can't handle the kind(%s) object", req.Kind.Kind),
			},
		}
	}

	if !mutationRequired(objectMeta) {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	var patch []patchOperation
	patch = append(patch, mutateYaml(objectMeta.GetAnnotations())...)

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		glog.Errorf("patch marshal error: %v", err)
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

func mutationRequired(metadata *metav1.ObjectMeta) bool {
	annotations := metadata.GetAnnotations()
	for k, _ := range annotations {
		if k == INSIGHT_CONNSTR_KEY || k == INSIGHT_ROLE_KEY {
			return true
		}
	}

	return false
}

func mutateYaml(content map[string]string) (patch []patchOperation) {
	for k, v := range content {
		patch = append(patch, patchOperation{
			Op:   "add",
			Path: "/spec/template/annotations",
			Value: map[string]string{
				k: v,
			},
		})
	}

	return
}
