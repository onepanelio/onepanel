<img width="240px" src="img/logo.png">

![build](https://img.shields.io/github/workflow/status/onepanelio/core/Publish%20dev%20docker%20image/master?color=01579b)
![code](https://img.shields.io/codacy/grade/d060fc4d1ac64b85b78f85c691ead86a?color=01579b)
[![release](https://img.shields.io/github/v/release/onepanelio/core?color=01579b)](https://github.com/onepanelio/core/releases)
[![sdk](https://img.shields.io/pypi/v/onepanel-sdk?color=01579b&label=sdk)](https://pypi.org/project/onepanel-sdk/)
[![docs](https://img.shields.io/github/v/release/onepanelio/core?color=01579b&label=docs)](https://docs.onepanel.io)
[![issues](https://img.shields.io/github/issues-raw/onepanelio/core?color=01579b&label=issues)](https://github.com/onepanelio/core/issues)
[![chat](https://img.shields.io/badge/support-slack-01579b)](https://join.slack.com/t/onepanel-ce/shared_invite/zt-eyjnwec0-nLaHhjif9Y~gA05KuX6AUg)
[![license](https://img.shields.io/github/license/onepanelio/core?color=01579b)](https://opensource.org/licenses/Apache-2.0)

Production scale vision AI platform with fully integrated components for model building, automated labeling, data processing and model training pipelines.

<img width="100%" src="img/onepanel.gif">

## Why Onepanel?

-  End-to-end automation for production scale vision AI pipelines
-  Best of breed, open source deep learning tools seamlessly integrated in one unified platform
-  Infrastructure automation so you can easily scale your data processing and training pipelines to multiple nodes
-  Customizable, reproducible and version controlled tooling and pipeline templates
-  Scalability, flexibility and resiliency of Kubernetes without the deployment and configuration complexities

## Features
-  Annotate images and video with automatic annotation of bounding boxes and polygon masks, fully integrated with data processing and training pipelines.
-  JupyterLab configured with extensions for TensorBoard, Git/GitHub, debugging, notebook diffing and support for Conda, OpenCV, Tensorflow and PyTorch with GPU.
-  Build fully reproducible, distributed and parallel data processing and training pipelines with real-time logs and output snapshots.
-  Bring your own IDEs, annotation tools and pipelines with a version controlled YAML and Docker based template engine.
-  Track and visualize model metrics and experiments with TensorBoard or bring your own experiment tracking tools.
-  Extend Onepanel with powerful REST APIs and SDKs to further automate your workflows.

## Online demo
We have created an [online demo environment](https://onepanel.typeform.com/to/kQfDX5Vf?product=github) so that you can quickly try Onepanel.

Note that this is a shared demo environment with the following restrictions:

- Data is reset every few hours
- One type of node pool (machine type) with a limit of 5 concurrent nodes
- Certain actions may be restricted

## Quick start
See [quick start guide](https://docs.onepanel.ai/docs/getting-started/quickstart) to get started with the platform of your choice.

### Quick start videos
[Getting started with Microsoft Azure](https://youtu.be/CQBIYfBk3Zk)\
[Getting started with Amazon EKS](https://youtu.be/Ipdd8f6D6IM)\
[Getting started with Google GKE](https://youtu.be/pZRO63SnQ8A)

## Community
See [documentation](https://docs.onepanel.ai) to get started or for more detailed operational and user guides.

To submit a feature request, report a bug or documentation issue, please open a GitHub [pull request](https://github.com/onepanelio/core/pulls) or [issue](https://github.com/onepanelio/core/issues).

For help, questions, release announcements and contribution discussions, join us on [Slack](https://join.slack.com/t/onepanel-ce/shared_invite/zt-eyjnwec0-nLaHhjif9Y~gA05KuX6AUg) or [GitHub discussions](https://github.com/onepanelio/core/discussions).

## Contributing

Onepanel is modular and consists of the following repositories:

[Backend](https://github.com/onepanelio/core/) (this repository) - Code base for backend (Go)\
[Frontend](https://github.com/onepanelio/core-ui/) - Code base for frontend (Angular + TypeScript)\
[CLI](https://github.com/onepanelio/cli/) - Code base for installation and management CLI (Go)\
[Manifests](https://github.com/onepanelio/core-ui/) - Kustomize manifests used by installation and management CLI (YAML)\
[Python SDK](https://github.com/onepanelio/python-sdk/) - Python SDK code and documentation (Python)\
[Templates](https://github.com/onepanelio/templates) - Various Workspace, Workflow, Task and Sidecar Templates\
[Documentation](https://github.com/onepanelio/core-docs/) - The repository for documentation site\
[API Documentation](https://github.com/onepanelio/core-api-docs/) - API documentation if you choose to use the API directly

See `CONTRIBUTING.md` in each repository for development guidelines. Also, see [contribution guide](https://docs.onepanel.ai/docs/getting-started/contributing) for additional guidelines.


## Acknowledgments
Onepanel seamlessly integrates the following excellent open source projects. We are grateful for the support these communities provide and do our best to contribute back as much as possible.

[Argo](https://github.com/argoproj/argo)\
[CVAT](https://github.com/opencv/cvat)\
[JupyterLab](https://github.com/jupyterlab/jupyterlab)\
[NNI](https://github.com/microsoft/nni)

## License
Onepanel is licensed under [Apache 2.0](https://github.com/onepanelio/core/blob/master/LICENSE).

## Need a managed solution?
Visit our [website](https://www.onepanel.io/) for more information about our managed offerings.
