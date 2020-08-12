package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/pressly/goose"
	"log"
)

const maskRCNNWorkflowTemplate = `arguments:
  parameters:
  - name: source
    value: https://github.com/onepanelio/Mask_RCNN.git
  - name: sys-annotation-path
    value: annotation-dump/sample_dataset
    hint: Please enter path which exists in cloud storage (i.e s3). In CVAT, this will be pre-populated. Some CVAT features may not work properly if you change that.
    displayName: Dataset path
  - name: sys-output-path
    value: workflow-data/output/sample_output
    hint: If you want to use this output in CVAT, it must have '<namespace>/workflow-data/output/<dir-name>' prefix.
    displayName: Workflow output path
  - name: extras
    value: none
  - name: num-classes
    value: 2
    hint: "Number of classes you have (i.e in CVAT task) +  1 for Background class."
    displayName: Number of classes
  - name: sys-finetune-checkpoint
    value: ""
    hint: "Default value is for demo purpose only. Please enter path which exists in cloud storage (i.e s3) or leave it empty."
    displayName: Checkpoint path
  - name: stage-1-epochs
    value: 1
    displayName: Epochs for network heads
  - name: stage-2-epochs
    value: 2
    displayName: Epochs for finetune layers (ResNet)
  - name: stage-3-epochs
    value: 3
    displayName: Epochs for finetuning all layers
  - name: tf-image
    value: tensorflow/tensorflow:1.13.1-py3
    type: select.select
    displayName: Select tensorflow image
    options:
    - name: 'Tensorflow 1.13.1 CPU Image'
      value: 'tensorflow/tensorflow:1.13.1-py3'
    - name: 'Tensorflow 1.13.1 GPU Image'
      value: 'tensorflow/tensorflow:1.13.1-gpu-py3'
  - displayName: Node pool
    hint: Name of node pool or group
    type: select.select
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
        --model=workflow_maskrcnn --stage1_epochs={{workflow.parameters.stage-1-epochs}} \
        --stage2_epochs={{workflow.parameters.stage-2-epochs}} \
        --stage3_epochs={{workflow.parameters.stage-3-epochs}} \
        --num_classes={{workflow.parameters.num-classes}} \
        --extras="{{workflow.parameters.extras}}"  \
        --ref_model_path="{{workflow.parameters.sys-finetune-checkpoint}}"  \
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
        storage: 200Gi
`

const maskRCNNWorkflowTemplateName = "MaskRCNN Training"

const tensorflowWorkflowTemplate2 = `arguments:
  parameters:
  - name: source
    value: https://github.com/tensorflow/models.git
  - name: trainingsource
    value: https://github.com/onepanelio/cvat-training.git
  - name: revision
    value: v1.13.0
  - name: sys-annotation-path
    value: annotation-dump/sample_dataset
    displayName: Dataset path
    hint: Please enter path which exists in cloud storage (i.e s3). In CVAT, this will be pre-populated. Some CVAT features may not work properly if you change that.
  - name: sys-output-path
    value: workflow-data/output/sample_output
    hint: If you want to use this output in CVAT, it must have '<namespace>/workflow-data/output/<dir-name>' prefix.
    displayName: Workflow output path
  - name: ref-model
    value: frcnn-res50-coco
    displayName: Reference model
    hint: Detection API's model to use for training.
    type: select.select
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
    value: '"epochs"="1000","schedule_step_1"=15000,"schedule_step_2"=18000'
  - name: sys-finetune-checkpoint
    value: ''
    hint: Please enter path which exists in cloud storage (i.e s3) or leave it empty.
    displayName: Checkpoint path
  - name: num-classes
    value: '5'
    displayName: Number of classes
  - name: tf-image
    value: tensorflow/tensorflow:1.13.1-py3
    type: select.select
    displayName: Select tensorflow image
    options:
    - name: 'Tensorflow 1.13.1 CPU Image'
      value: 'tensorflow/tensorflow:1.13.1-py3'
    - name: 'Tensorflow 1.13.1 GPU Image'
      value: 'tensorflow/tensorflow:1.13.1-gpu-py3'
  - displayName: Node pool
    hint: Name of node pool or group
    type: select.select
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
      apt-get update && \
      apt-get install -y python3-pip git wget unzip libglib2.0-0 libsm6 libxext6 libxrender-dev && \
      pip install pillow lxml Cython contextlib2 jupyter matplotlib numpy scipy boto3 pycocotools pyyaml google-cloud-storage && \
      cd /mnt/src/tf/research && \
      export PYTHONPATH=$PYTHONPATH:` + "`pwd`:`pwd`/slim &&" + `\
      cd /mnt/src/train && \
      python convert_workflow.py {{workflow.parameters.extras}},dataset=/mnt/data/datasets,model={{workflow.parameters.ref-model}},num_classes={{workflow.parameters.num-classes}},sys-finetune-checkpoint={{workflow.parameters.sys-finetune-checkpoint}}
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
        revision: '{{workflow.parameters.revision}}'
      name: src
      path: /mnt/src/tf
    - git:
        repo: '{{workflow.parameters.trainingsource}}'
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
        storage: 200Gi
`

func initialize20200812104328() {
	if _, ok := initializedMigrations[20200812104328]; !ok {
		goose.AddMigration(Up20200812104328, Down20200812104328)
		initializedMigrations[20200812104328] = true
	}
}

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
	}
	if err := workflowTemplate.GenerateUID(workflowTemplate.Name); err != nil {
		return err
	}

	for _, namespace := range namespaces {
		existingWorkflowTemplate, err := client.GetLatestWorkflowTemplate(namespace.Name, workflowTemplate.UID)
		if err != nil {
			return err
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
		Name:     tensorflowWorkflowTemplateName,
		Manifest: tensorflowWorkflowTemplate2,
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

func Down20200812104328(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
