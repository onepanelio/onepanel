package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/pressly/goose"
	"log"
	"strings"
)

const maskRCNNWorkflowTemplate = `arguments:
  parameters:
  - name: source
    value: https://github.com/onepanelio/Mask_RCNN.git
    displayName: Model source code
    type: hidden
    visibility: private

  - name: sys-annotation-path
    value: annotation-dump/sample_dataset
    hint: Path to annotated data in default object storage (i.e S3). In CVAT, this parameter will be pre-populated.
    displayName: Dataset path
    visibility: private
    
  - name: sys-output-path
    value: workflow-data/output/sample_output
    hint: Path to store output artifacts in default object storage (i.e s3). In CVAT, this parameter will be pre-populated.
    displayName: Workflow output path
    visibility: private

  - name: sys-finetune-checkpoint
    value: ''
    hint: Select the last fine-tune checkpoint for this model. It may take up to 5 minutes for a recent checkpoint show here. Leave empty if this is the first time you're training this model.
    displayName: Checkpoint path
    visibility: public
  
  - name: sys-num-classes
    displayName: Number of classes
    hint: Number of classes (i.e in CVAT taks) + 1 for background
    value: 81
    visibility: private
    
  - name: extras
    displayName: Hyperparameters
    visibility: public
    type: textarea.textarea
    value: |-
      stage-1-epochs=1    #  Epochs for network heads
      stage-2-epochs=2    #  Epochs for finetune layers
      stage-3-epochs=3    #  Epochs for all layers
    hint: "Please refer to our <a href='https://docs.onepanel.ai/docs/getting-started/use-cases/computervision/annotation/cvat/cvat_annotation_model#arguments-optional' target='_blank'>documentation</a> for more information on parameters. Number of classes will be automatically populated if you had 'sys-num-classes' parameter in a workflow."
    
  - name: dump-format
    type: select.select
    value: cvat_coco
    displayName: CVAT dump format
    visibility: public
    options:
    - name: 'MS COCO'
      value: 'cvat_coco'
    - name: 'TF Detection API'
      value: 'cvat_tfrecord'
      
  - name: tf-image
    visibility: public
    value: tensorflow/tensorflow:1.13.1-py3
    type: select.select
    displayName: Select TensorFlow image
    hint: Select the GPU image if you are running on a GPU node pool
    options:
    - name: 'TensorFlow 1.13.1 CPU Image'
      value: 'tensorflow/tensorflow:1.13.1-py3'
    - name: 'TensorFlow 1.13.1 GPU Image'
      value: 'tensorflow/tensorflow:1.13.1-gpu-py3'
  
  - displayName: Node pool
    hint: Name of node pool or group to run this workflow task
    type: select.select
    visibility: public
    name: sys-node-pool
    value: Standard_D4s_v3
    required: true
    options:
    - name: 'CPU: 2, RAM: 8GB'
      value: Standard_D2s_v3
    - name: 'CPU: 4, RAM: 16GB'
      value: Standard_D4s_v3
    - name: 'GPU: 1xK80, CPU: 6, RAM: 56GB'
      value: Standard_NC6

entrypoint: main
templates:
- dag:
    tasks:
    - name: train-model
      template: tensorflow
# Uncomment the lines below if you want to send Slack notifications
#    - arguments:
#        artifacts:
#        - from: '{{tasks.train-model.outputs.artifacts.sys-metrics}}'
#          name: metrics
#        parameters:
#        - name: status
#          value: '{{tasks.train-model.status}}'
#      dependencies:
#      - train-model
#      name: notify-in-slack
#      template: slack-notify-success
  name: main
- container:
    args:
    - |
      apt-get update \
      && apt-get install -y git wget libglib2.0-0 libsm6 libxext6 libxrender-dev \
      && pip install -r requirements.txt \
      && pip install boto3 pyyaml google-cloud-storage \
      && git clone https://github.com/waleedka/coco \
      && cd coco/PythonAPI \
      && python setup.py build_ext install \
      && rm -rf build \
      && cd ../../ \
      && wget https://github.com/matterport/Mask_RCNN/releases/download/v2.0/mask_rcnn_coco.h5 \
      && python setup.py install && ls \
      && python samples/coco/cvat.py train --dataset=/mnt/data/datasets \
        --model=workflow_maskrcnn \
        --extras="{{workflow.parameters.extras}}"  \
        --ref_model_path="{{workflow.parameters.sys-finetune-checkpoint}}"  \
        --num_classes="{{workflow.parameters.sys-num-classes}}" \
      && cd /mnt/src/ \
      && python prepare_dataset.py /mnt/data/datasets/annotations/instances_default.json
    command:
    - sh
    - -c
    image: '{{workflow.parameters.tf-image}}'
    volumeMounts:
    - mountPath: /mnt/data
      name: data
    - mountPath: /mnt/output
      name: output
    workingDir: /mnt/src
  nodeSelector:
    beta.kubernetes.io/instance-type: '{{workflow.parameters.sys-node-pool}}'
  inputs:
    artifacts:
    - name: data
      path: /mnt/data/datasets/
      s3:
        key: '{{workflow.namespace}}/{{workflow.parameters.sys-annotation-path}}'
    - git:
        repo: '{{workflow.parameters.source}}'
        revision: "no-boto"
      name: src
      path: /mnt/src
  name: tensorflow
  outputs:
    artifacts:
    - name: model
      optional: true
      path: /mnt/output
      s3:
        key: '{{workflow.namespace}}/{{workflow.parameters.sys-output-path}}'
# Uncomment the lines below if you want to send Slack notifications
#- container:
#    args:
#    - SLACK_USERNAME=Onepanel SLACK_TITLE="{{workflow.name}} {{inputs.parameters.status}}"
#      SLACK_ICON=https://www.gravatar.com/avatar/5c4478592fe00878f62f0027be59c1bd
#      SLACK_MESSAGE=$(cat /tmp/metrics.json)} ./slack-notify
#    command:
#    - sh
#    - -c
#    image: technosophos/slack-notify
#  inputs:
#    artifacts:
#    - name: metrics
#      optional: true
#      path: /tmp/metrics.json
#    parameters:
#    - name: status
#  name: slack-notify-success
volumeClaimTemplates:
- metadata:
    creationTimestamp: null
    name: data
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 200Gi
- metadata:
    creationTimestamp: null
    name: output
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 200Gi`

const maskRCNNWorkflowTemplateName = "MaskRCNN Training"

const tensorflowObjectDetectionWorkflowTemplate = `arguments:
  parameters:
  - name: source
    value: https://github.com/tensorflow/models.git
    displayName: Model source code
    type: hidden
    visibility: private

  - name: trainingsource
    value: https://github.com/onepanelio/cvat-training.git
    type: hidden
    visibility: private

  - name: revision
    value: v1.13.0
    type: hidden 
    visibility: private

  - name: sys-annotation-path
    value: annotation-dump/sample_dataset
    displayName: Dataset path
    hint: Path to annotated data in default object storage (i.e S3). In CVAT, this parameter will be pre-populated.

  - name: sys-output-path
    value: workflow-data/output/sample_output
    hint: Path to store output artifacts in default object storage (i.e s3). In CVAT, this parameter will be pre-populated.
    displayName: Workflow output path
    visibility: private

  - name: ref-model
    value: frcnn-res50-coco
    displayName: Model
    hint: TF Detection API's model to use for training.
    type: select.select
    visibility: public
    options:
    - name: 'Faster RCNN-ResNet 101-COCO'
      value: frcnn-res101-coco
    - name: 'Faster RCNN-ResNet 101-Low Proposal-COCO'
      value: frcnn-res101-low
    - name: 'Faster RCNN-ResNet 50-COCO'
      value: frcnn-res50-coco
    - name: 'Faster RCNN-NAS-COCO'
      value: frcnn-nas-coco
    - name: 'SSD MobileNet V1-COCO'
      value: ssd-mobilenet-v1-coco2
    - name: 'SSD MobileNet V2-COCO'
      value: ssd-mobilenet-v2-coco
    - name: 'SSDLite MobileNet-COCO'
      value: ssdlite-mobilenet-coco
  
  - name: extras
    value: |-
      epochs=1000
    displayName: Hyperparameters
    visibility: public
    type: textarea.textarea
    hint: "Please refer to our <a href='https://docs.onepanel.ai/docs/getting-started/use-cases/computervision/annotation/cvat/cvat_annotation_model#arguments-optional' target='_blank'>documentation</a> for more information on parameters. Number of classes will be automatically populated if you had 'sys-num-classes' parameter in a workflow."
  
  - name: sys-finetune-checkpoint
    value: ''
    hint: Select the last fine-tune checkpoint for this model. It may take up to 5 minutes for a recent checkpoint show here. Leave empty if this is the first time you're training this model.
    displayName: Checkpoint path
    visibility: public
    
  - name: sys-num-classes
    value: 81
    hint: Number of classes
    displayName: Number of classes
    visibility: private

  - name: tf-image
    value: tensorflow/tensorflow:1.13.1-py3
    type: select.select
    displayName: Select TensorFlow image
    visibility: public
    hint: Select the GPU image if you are running on a GPU node pool
    options:
    - name: 'TensorFlow 1.13.1 CPU Image'
      value: 'tensorflow/tensorflow:1.13.1-py3'
    - name: 'TensorFlow 1.13.1 GPU Image'
      value: 'tensorflow/tensorflow:1.13.1-gpu-py3'

  - displayName: Node pool
    hint: Name of node pool or group to run this workflow task
    type: select.select
    name: sys-node-pool
    value: Standard_D4s_v3
    visibility: public
    required: true
    options:
    - name: 'CPU: 2, RAM: 8GB'
      value: Standard_D2s_v3
    - name: 'CPU: 4, RAM: 16GB'
      value: Standard_D4s_v3
    - name: 'GPU: 1xK80, CPU: 6, RAM: 56GB'
      value: Standard_NC6
  - name: dump-format
    value: cvat_tfrecord
    visibility: public
entrypoint: main
templates:
- dag:
    tasks:
    - name: train-model
      template: tensorflow
# Uncomment the lines below if you want to send Slack notifications
#    - arguments:
#        artifacts:
#        - from: '{{tasks.train-model.outputs.artifacts.sys-metrics}}'
#          name: metrics
#        parameters:
#        - name: status
#          value: '{{tasks.train-model.status}}'
#      dependencies:
#      - train-model
#      name: notify-in-slack
#      template: slack-notify-success
  name: main
- container:
    args:
    - |
      apt-get update && \
      apt-get install -y python3-pip git wget unzip libglib2.0-0 libsm6 libxext6 libxrender-dev && \
      pip install pillow lxml Cython contextlib2 jupyter matplotlib numpy scipy boto3 pycocotools pyyaml google-cloud-storage && \
      cd /mnt/src/tf/research && \
      export PYTHONPATH=$PYTHONPATH:` + "`pwd`:`pwd`/slim" + ` && \
      cd /mnt/src/train && \
      python convert_workflow.py \
        --extras="{{workflow.parameters.extras}}" \
        --model="{{workflow.parameters.ref-model}}" \
        --num_classes="{{workflow.parameters.sys-num-classes}}" \
        --sys_finetune_checkpoint={{workflow.parameters.sys-finetune-checkpoint}}
    command:
    - sh
    - -c
    image: '{{workflow.parameters.tf-image}}'
    volumeMounts:
    - mountPath: /mnt/data
      name: data
    - mountPath: /mnt/output
      name: output
    workingDir: /mnt/src
  nodeSelector:
    beta.kubernetes.io/instance-type: '{{workflow.parameters.sys-node-pool}}'
  inputs:
    artifacts:
    - name: data
      path: /mnt/data/datasets/
      s3:
        key: '{{workflow.namespace}}/{{workflow.parameters.sys-annotation-path}}'
    - name: models
      path: /mnt/data/models/
      optional: true
      s3:
        key: '{{workflow.namespace}}/{{workflow.parameters.sys-finetune-checkpoint}}'
    - git:
        repo: '{{workflow.parameters.source}}'
        revision: '{{workflow.parameters.revision}}'
      name: src
      path: /mnt/src/tf
    - git:
        repo: '{{workflow.parameters.trainingsource}}'
        revision: 'optional-artifacts'
      name: tsrc
      path: /mnt/src/train
  name: tensorflow
  outputs:
    artifacts:
    - name: model
      optional: true
      path: /mnt/output
      s3:
        key: '{{workflow.namespace}}/{{workflow.parameters.sys-output-path}}'
# Uncomment the lines below if you want to send Slack notifications
#- container:
#    args:
#    - SLACK_USERNAME=Onepanel SLACK_TITLE="{{workflow.name}} {{inputs.parameters.status}}"
#      SLACK_ICON=https://www.gravatar.com/avatar/5c4478592fe00878f62f0027be59c1bd
#      SLACK_MESSAGE=$(cat /tmp/metrics.json)} ./slack-notify
#    command:
#    - sh
#    - -c
#    image: technosophos/slack-notify
#  inputs:
#    artifacts:
#    - name: metrics
#      optional: true
#      path: /tmp/metrics.json
#    parameters:
#    - name: status
#  name: slack-notify-success
volumeClaimTemplates:
- metadata:
    creationTimestamp: null
    name: data
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 200Gi
- metadata:
    creationTimestamp: null
    name: output
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 200Gi`

const tensorflowObjectDetectionWorkflowTemplateName = "TensorFlow Object Detection Training"

func initialize20200812104328() {
	if _, ok := initializedMigrations[20200812104328]; !ok {
		goose.AddMigration(Up20200812104328, Down20200812104328)
		initializedMigrations[20200812104328] = true
	}
}

// Up20200812104328 runs the migration to update MaskRCNN and TF_OD templates
func Up20200812104328(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	migrationsRan, err := getRanSQLMigrations(client)
	if err != nil {
		return err
	}

	if _, ok := migrationsRan[20200812104328]; ok {
		return nil
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	// Create maskrcnn
	workflowTemplate := &v1.WorkflowTemplate{
		Name:     maskRCNNWorkflowTemplateName,
		Manifest: maskRCNNWorkflowTemplate,
		Labels: map[string]string{
			"used-by": "cvat",
		},
	}
	if err := workflowTemplate.GenerateUID(workflowTemplate.Name); err != nil {
		return err
	}

	for _, namespace := range namespaces {
		existingWorkflowTemplate, err := client.GetLatestWorkflowTemplate(namespace.Name, workflowTemplate.UID)
		if err != nil {
			if strings.Contains(err.Error(), "Workflow template not found") {
				err = nil
				existingWorkflowTemplate = nil
			} else {
				return err
			}
		}
		if existingWorkflowTemplate != nil {
			log.Printf("Skipping creating template '%v'. It already exists in namespace '%v'", workflowTemplate.Name, namespace.Name)
			continue
		}

		if _, err := client.CreateWorkflowTemplate(namespace.Name, workflowTemplate); err != nil {
			return err
		}
	}

	// Update tf-od
	workflowTemplate = &v1.WorkflowTemplate{
		Name:     tensorflowObjectDetectionWorkflowTemplateName,
		Manifest: tensorflowObjectDetectionWorkflowTemplate,
		Labels: map[string]string{
			"used-by": "cvat",
		},
	}
	if err := workflowTemplate.GenerateUID(workflowTemplate.Name); err != nil {
		return err
	}

	for _, namespace := range namespaces {
		if _, err := client.CreateWorkflowTemplateVersion(namespace.Name, workflowTemplate); err != nil {
			return err
		}
	}

	return nil
}

// Down20200812104328 does nothing
func Down20200812104328(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
