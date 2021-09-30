package datasource

import (
	"github.com/go-errors/errors"
	"github.com/redhatinsights/xjoin-operator/api/v1alpha1"
	"github.com/redhatinsights/xjoin-operator/controllers/common"
	"github.com/redhatinsights/xjoin-operator/controllers/parameters"
	"github.com/redhatinsights/xjoin-operator/controllers/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const XJOIN_COMPONENT_NAME_LABEL = "xjoin.component.name"

type XJoinDataSourceIteration struct {
	common.Iteration
	Parameters parameters.DataSourceParameters
}

var dataSourcePipelineGVK = schema.GroupVersionKind{
	Group:   "xjoin.cloud.redhat.com",
	Kind:    "XJoinDataSourcePipeline",
	Version: "v1alpha1",
}

func (i *XJoinDataSourceIteration) CreateDataSourcePipeline(name string, version string) (err error) {
	dataSourcePipeline := unstructured.Unstructured{}
	dataSourcePipeline.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      name + "." + version,
			"namespace": i.Iteration.Instance.GetNamespace(),
			"labels": map[string]interface{}{
				XJOIN_COMPONENT_NAME_LABEL: name,
			},
		},
		"spec": map[string]interface{}{
			"name":             name,
			"version":          version,
			"avroSchema":       i.Parameters.AvroSchema.String(),
			"databaseHostname": i.Parameters.DatabaseHostname.String(),
			"databasePort":     i.Parameters.DatabasePort.String(),
			"databaseName":     i.Parameters.DatabaseName.String(),
			"databaseUsername": i.Parameters.DatabaseUsername.String(),
			"databasePassword": i.Parameters.DatabasePassword.String(),
			"pause":            i.Parameters.Pause.Bool(),
		},
	}
	dataSourcePipeline.SetGroupVersionKind(dataSourcePipelineGVK)
	err = i.CreateChildResource(dataSourcePipeline)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	return
}

func (i *XJoinDataSourceIteration) DeleteDataSourcePipeline(name string, version string) (err error) {
	err = i.DeleteResource(name+"."+version, dataSourcePipelineGVK)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	return
}

func (i *XJoinDataSourceIteration) ReconcilePipelines() (err error) {
	//build an array and map of expected pipeline versions (active, refreshing)
	//the map value will be set to true when an expected pipeline is found
	expectedPipelinesMap := make(map[string]bool)
	var expectedPipelinesArray []string
	if i.GetInstance().Status.ActiveVersion != "" {
		expectedPipelinesMap[i.GetInstance().Status.ActiveVersion] = false
		expectedPipelinesArray = append(expectedPipelinesArray, i.GetInstance().Status.ActiveVersion)
	}
	if i.GetInstance().Status.RefreshingVersion != "" {
		expectedPipelinesMap[i.GetInstance().Status.RefreshingVersion] = false
		expectedPipelinesArray = append(expectedPipelinesArray, i.GetInstance().Status.RefreshingVersion)
	}

	//retrieve a list of pipelines for this datasource.name
	pipelines := &unstructured.UnstructuredList{}
	pipelines.SetGroupVersionKind(dataSourcePipelineGVK)
	labels := client.MatchingLabels{}
	labels[XJOIN_COMPONENT_NAME_LABEL] = i.GetInstance().Name
	err = i.Client.List(i.Context, pipelines, labels)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	//remove any extra pipelines, ensure the expected pipelines are created
	for _, pipeline := range pipelines.Items {
		spec := pipeline.Object["spec"].(map[string]interface{})
		pipelineVersion := spec["version"].(string)
		if !utils.ContainsString(expectedPipelinesArray, pipelineVersion) {
			err = i.DeleteDataSourcePipeline(i.GetInstance().Name, pipelineVersion)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		} else {
			expectedPipelinesMap[pipelineVersion] = true
		}
	}

	for version, exists := range expectedPipelinesMap {
		if !exists {
			i.Log.Info("expected pipeline version " + version + " not found, recreating it")
			err = i.CreateDataSourcePipeline(i.GetInstance().Name, version)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	return
}

func (i *XJoinDataSourceIteration) Finalize() (err error) {
	i.Log.Info("Starting finalizer")
	if i.GetInstance().Status.ActiveVersion != "" {
		err = i.DeleteDataSourcePipeline(i.GetInstance().Name, i.GetInstance().Status.ActiveVersion)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}
	if i.GetInstance().Status.RefreshingVersion != "" {
		err = i.DeleteDataSourcePipeline(i.GetInstance().Name, i.GetInstance().Status.RefreshingVersion)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	controllerutil.RemoveFinalizer(i.Instance, i.GetFinalizerName())
	ctx, cancel := utils.DefaultContext()
	defer cancel()
	err = i.Client.Update(ctx, i.Instance)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	i.Log.Info("Successfully finalized")
	return nil
}

func (i XJoinDataSourceIteration) GetInstance() *v1alpha1.XJoinDataSource {
	return i.Instance.(*v1alpha1.XJoinDataSource)
}

func (i XJoinDataSourceIteration) GetFinalizerName() string {
	return "finalizer.xjoin.datasource.cloud.redhat.com"
}
