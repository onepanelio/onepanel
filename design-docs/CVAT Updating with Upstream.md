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

# Copy everything over `cvat/apps/onepanelio`
![](.CVAT Updating with Upstream_images/7a2f91b9.png)

# Add relevant onepanel URLs to `cvat/urls.py`
![](.CVAT Updating with Upstream_images/e9122ada.png)

# Update `cvat/settings/base.py` to enable onepanel related pieces.
![](.CVAT Updating with Upstream_images/147082c5.png)
![](.CVAT Updating with Upstream_images/02441b19.png)

# As of right now, we need to enable environment variables for CVAT as well.
`cvat/settings/base.py`
![](.CVAT Updating with Upstream_images/92b76b3c.png)
![](.CVAT Updating with Upstream_images/dead64db.png)