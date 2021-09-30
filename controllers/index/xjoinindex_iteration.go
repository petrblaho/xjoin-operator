package index

import (
	"github.com/redhatinsights/xjoin-operator/api/v1alpha1"
	"github.com/redhatinsights/xjoin-operator/controllers/common"
	"github.com/redhatinsights/xjoin-operator/controllers/parameters"
	"github.com/redhatinsights/xjoin-operator/controllers/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type XJoinIndexIteration struct {
	common.Iteration
	Parameters parameters.IndexParameters
}

var indexPipelineGVK = schema.GroupVersionKind{
	Group:   "xjoin.cloud.redhat.com",
	Kind:    "XJoinIndexPipeline",
	Version: "v1alpha1",
}

var indexValidatorGVK = schema.GroupVersionKind{
	Group:   "xjoin.cloud.redhat.com",
	Kind:    "XJoinIndexValidator",
	Version: "v1alpha1",
}

func (i *XJoinIndexIteration) CreateIndexPipeline(name string, version string) (err error) {
	dataSourcePipeline := unstructured.Unstructured{}
	dataSourcePipeline.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      name + "." + version,
			"namespace": i.Iteration.Instance.GetNamespace(),
		},
		"spec": map[string]interface{}{
			"version":    version,
			"avroSchema": i.Parameters.AvroSchema.String(),
			"pause":      i.Parameters.Pause.Bool(),
		},
	}
	dataSourcePipeline.SetGroupVersionKind(indexPipelineGVK)

	return i.CreateChildResource(dataSourcePipeline)
}

func (i *XJoinIndexIteration) CreateIndexValidator(name string, version string) (err error) {
	dataSourcePipeline := unstructured.Unstructured{}
	dataSourcePipeline.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      name + "." + version,
			"namespace": i.Iteration.Instance.GetNamespace(),
		},
		"spec": map[string]interface{}{
			"version":    version,
			"avroSchema": i.Parameters.AvroSchema.String(),
			"pause":      i.Parameters.Pause.Bool(),
		},
	}
	dataSourcePipeline.SetGroupVersionKind(indexValidatorGVK)
	return i.CreateChildResource(dataSourcePipeline)
}

func (i *XJoinIndexIteration) DeleteIndexPipeline(name string, version string) (err error) {
	return i.DeleteResource(name+"."+version, indexPipelineGVK)
}

func (i *XJoinIndexIteration) DeleteIndexValidator(name string, version string) (err error) {
	return i.DeleteResource(name+"."+version, indexValidatorGVK)
}

func (i XJoinIndexIteration) GetInstance() *v1alpha1.XJoinIndex {
	return i.Instance.(*v1alpha1.XJoinIndex)
}

func (i XJoinIndexIteration) GetFinalizerName() string {
	return "finalizer.xjoin.index.cloud.redhat.com"
}

func (i *XJoinIndexIteration) Finalize() (err error) {
	i.Log.Info("Starting finalizer")
	controllerutil.RemoveFinalizer(i.Iteration.Instance, i.GetFinalizerName())

	ctx, cancel := utils.DefaultContext()
	defer cancel()
	err = i.Client.Update(ctx, i.Iteration.Instance)
	if err != nil {
		return
	}

	i.Log.Info("Successfully finalized")
	return nil
}
