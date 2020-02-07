👍🎉 First off, thanks for taking the time to contribute! 🎉👍

# Roadmap
Coming Soon

# Code
**General Guidelines**

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

# Coding Conventions
This repository uses lots of golang and yaml.
- We expect you to run `go fmt` before a PR.
- We also expect you to run a go linter to check for issues.

# Issues

Did you find a bug?

- Do not open up a GitHub issue if the bug is a security vulnerability in Go lang.

- Ensure the bug was not already reported by searching on GitHub under Issues.

- If you're unable to find an open issue addressing the problem, open a new one. Be sure to include a **title and clear description**, as much relevant information as possible, and a **code sample** or an **executable test case** demonstrating the expected behavior that is not occurring.

Did you write a patch that fixes a bug?

- Open a new GitHub pull request with the patch.

- Ensure the PR description clearly describes the problem and solution. Include the relevant issue number if applicable.

# Pull Requests

Always write a clear log message for your commits. One-line messages are fine for small changes, but bigger changes should look like this:

```
$ git commit -m "A brief summary of the commit
> 
> A paragraph describing what changed and its impact."
```

Run `go fmt` on your entire project.
# Behavioral Expectations

