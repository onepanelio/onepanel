<img width="240px" src="img/logo.png">

![build](https://img.shields.io/github/workflow/status/onepanelio/core/Build%20and%20publish%20to%20Docker%20Hub/master?color=01579b)
![code](https://img.shields.io/codacy/grade/d060fc4d1ac64b85b78f85c691ead86a?color=01579b)
[![release](https://img.shields.io/github/v/release/onepanelio/core?color=01579b)](https://github.com/onepanelio/core/releases)
[![sdk](https://img.shields.io/pypi/v/onepanel-sdk?color=01579b&label=sdk)](https://pypi.org/project/onepanel-sdk/)
[![docs](https://img.shields.io/github/v/release/onepanelio/core?color=01579b&label=docs)](https://docs.onepanel.io)
[![issues](https://img.shields.io/github/issues-raw/onepanelio/core?color=01579b&label=issues)](https://github.com/onepanelio/core/issues)
[![chat](https://img.shields.io/badge/support-slack-01579b)](https://join.slack.com/t/onepanel-ce/shared_invite/zt-eyjnwec0-nLaHhjif9Y~gA05KuX6AUg)
[![license](https://img.shields.io/github/license/onepanelio/core?color=01579b)](https://opensource.org/licenses/Apache-2.0)

Full stack vision AI platform with fully integrated modules for model building, semi-automated labeling, training and data pipelines, powered by Kubernetes.

## Features
| | |
|:-------------------------:|:-------------------------:|
|<img width="1604" src="img/auto-annotation.gif"> <h4>Image and video annotation with automatic pre-annotation and full integration with training pipelines</h4>|<img width="1604" src="img/jupyterlab.gif"> <h4>JupyterLab loaded with popular deep learning libraries like Tensorflow, PyTorch and TensorBoard with GPU support</h4>
|<img width="1604" src="img/pipelines.gif"> <h4>Build auto-scaling, distributed data processing and training pipelines with streaming logs and output snapshots</h4>|<img width="1604" src="img/tools.gif"> <h4>Bring your own IDEs, annotation tools and workflows using YAML and Docker based templating system</h4>|

## Quick start
See [quick start guide](https://docs.onepanel.ai/docs/getting-started/quickstart) to get started with the platform of your choice.

## Community


## Contributing

Onepanel consists of the following repositories:

[Core API](https://github.com/onepanelio/core/) (this repository) - Code base for backend (Go)\
[Core UI](https://github.com/onepanelio/core-ui/) - Code base for UI (Angular + TypeScript)\
[CLI](https://github.com/onepanelio/cli/) - Code base for Go CLI for installation and management (Go)\
[Manifests](https://github.com/onepanelio/core-ui/) - Kustomize manifests used by CLI for installation and management (YAML)\
[Python SDK](https://github.com/onepanelio/python-sdk/) - Python SDK code and documentation\
[Templates](https://github.com/onepanelio/templates) - Various Workspace, Workflow, Task and Sidecar Templates\
[Documentation](https://github.com/onepanelio/core-docs/) - The repository for documentation site\
[API Documentation](https://github.com/onepanelio/core-api-docs/) - API documentation if you choose to use the API directly

See `CONTRIBUTING.md` in each repository for development guidelines. Also, see [contribution guide](https://docs.onepanel.ai/docs/getting-started/contributing) for additional guidelines.

## Acknowledgments
We use these excellent open source projects to power different areas of Onepanel:

[Argo](https://github.com/argoproj/argo)\
[CVAT](https://github.com/opencv/cvat)\
[JupyterLab](https://github.com/jupyterlab/jupyterlab)
