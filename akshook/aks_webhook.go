package akshook

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kubernetes/pkg/apis/core/v1"
	"net/http"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

type WebhookServer struct {
	Server *http.Server
}

type AKSWebhookParameters struct {
	Port           int    // webhook server Port
	CertFile       string // path to the x509 certificate for https
	KeyFile        string // path to the x509 private key matching `CertFile`
	sidecarCfgFile string // path to sidecar injector configuration file
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	_ = v1.AddToScheme(runtimeScheme)
}

func yamlUpdater(addedPatch *patchOperation) ([]byte, error){
	var newYaml []patchOperation
	newYaml = append(newYaml, *addedPatch)
	return json.Marshal(newYaml)
}

func annotationSearcher(annotation map[string]string, key string, value string) bool{
	for k, v := range annotation{
		if k == key && v == value {
			return true
		}
	}

	return false
}

func (ws *WebhookServer) mutating(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse{
	req := ar.Request
	var annotations map[string]string
	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v UID=%v UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)

	switch req.Kind.Kind{
	case "Deployment":
		var deploy appsv1.Deployment

		if err := json.Unmarshal(req.Object.Raw, &deploy); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}

		annotations = deploy.GetAnnotations()

	case "Service":
		var service corev1.Service

		if err := json.Unmarshal(req.Object.Raw, &service); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}

		annotations = service.GetAnnotations()
	}
	
	patchTest := &patchOperation{
		Op:    "add",
		Path:  "/spec/template/metadata/annotations",
		Value: map[string]string{
			"AKS-YAMLUPDATER":"TRUE",
		},
	}

	if !annotationSearcher(annotations, "AKS-Insight", "TRUE"){
		glog.Info("Annotation Not Found, Skip Mutation")
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	byteResult, err := yamlUpdater(patchTest)
	if err != nil {
		glog.Errorf("Patch Error at yamlUpdater, Message: %s", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   byteResult,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func (ws *WebhookServer) Serve(w http.ResponseWriter, r *http.Request){
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		fmt.Println(r.URL.Path)
		admissionResponse = ws.mutating(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write response ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}

}

