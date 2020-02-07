# Code
Files under `/kube/`, such as `/kube/workflow.go` are meant to deal with database, kubernetes,
or other data storage / repositories of raw information.
- Errors are returned directly. Usually to the manager.

Files under `/server/`, such as `/server/workflow_server.go` are meant to receive requests.
That code is generally meant to create Data Transfer Objects, and pass them along to
the relevant manager.
- If an error is returned from a manager, the server will pass it along as a specific GRPC error
type.

Files under `/manager` such as `/manager/workflow_manager.go` are meant to take the DTO
from server and carry out business logic / preparation of the data to pass along
to `/kube/`.
- If a kube file is doing a lot of work to prepare the data, such as setting fields
or figuring out if a value exists? That's a sign that the logic should be moved out
into the manager.
- The kube file should focus on taking a model of the data in, and returning a model with
the data.
- Sometimes, it may not make sense to generate a whole model, and in those cases, parameters
are fine.
- Errors encountered from kube, in the manager, are to be converted to a specific
error type with a "codes". To denote it's a "NotFound" or "Unathorized" error.
So that GRPC returns the correct HTTP code.

# Issues

# Pull Requests

# Behavioral Expectations

