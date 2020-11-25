# Assume all file paths are from project root.

# `Dockerfile`
Usually, we add vim and nano to the build for ease of debugging remotely (and inside
the CLI).
![](.CVAT Updating with Upstream_images/7037581f.png)
- Not required

# `.gitignore`
If you're using JetBrains IDE products, this will keep out their IDE folder.
![](.CVAT Updating with Upstream_images/9f162879.png)

# `cvat/requirements/base.txt`
Add onepanel-sdk and google-cloud-storage.
- SDK for our authentication and creating workflows on onepanel, from inside CVAT.
- Google Cloud Storage because we support uploading to GCS.
    - Storing annotations, etc
    
![](.CVAT Updating with Upstream_images/438146d0.png)