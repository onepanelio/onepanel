# Background
A user may install pip packages, conda packages, jupyterlab extensions,
or vscode extensions.

To improve the user experience, we should save these packages
when the user pauses and resumes the workspace.
- And when the user changes the workspace machine.

To achieve this end, we’re using lifecycle hooks for containers, via Kubernetes.

We’ll add these hooks to workspace templates, to VSCode and JupyterLab.

Note: We're not supporting CVAT Workspace persistence.
- User does not have access to the terminal through the UI.

# Problem(s)
Originally, we attempted to tar up the packages of conda, vscode, and jupyter.
This worked, but ate non-trivial amount of space. Roughly ~2 GB with a clean workspace,
using our dockerfile image.
- So we'd need to warn the user to have enough space for their volume mounts
- We'd also have to warn the user that if there is not enough space, the packages
may not persist.

The other problem was we needed to extend the terminationGracePeriodSeconds of the
pod.
- K8s sets 30 seconds by default

We use the preStop hook to do the back-up.
- This operation runs for 2-3 minutes on a clean workspace image
- For a user with more packages, it would take longer.
- k8s will grant a 2 second increase once, if preStop is not done.

Otherwise, k8s kills the preStop hook and the container.

We thought about exposing the terminationGracePeriodSeconds to the user,
allow them to change this value as needed.

And lastly, k8s will wait the entire grace period.
- Even if the preStop hook finishes early, k8s will wait to kill the container.
- In testing, k8s waited 20 minutes before terminating the container

So we decided to export the list of installed packages, as a text file.
- Then, we re-install on workspace start-up with postStart hook.

# References
- https://github.com/onepanelio/core/issues/623