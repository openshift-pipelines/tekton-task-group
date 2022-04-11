# Tekton `TaskGroup` Custom Task

Tekton [custom
task](https://github.com/tektoncd/pipeline/blob/main/docs/runs.md)
that allows to group Task together as a Task. It will merge `Tasks`
into one and run it as a `TaskRun` (with embedded spec). This is a
slightly different approach than "Pipeline in a pod"
([TEP-0044](https://github.com/tektoncd/community/blob/main/teps/0044-decouple-task-composition-from-scheduling.md)).

Most of the time, if we want to checkout, build, test and ship
something using TektonCD Pipeline, we are defining `Pipeline`. This,
however comes with some overhead like the following :

- Provisioning PVC to share data
- Multiple pod running

For simple cases, like checkout a source code and build an image, it
should be possible to run all in the same `Pod` as a single Task. The
idea would be to be able to "augment" a `Task` (at *runtime* or
*authoring* time) with additionnal steps.

## Install

To install the `TaskGroup` custom task, you will need
[`ko`](https://github.com/google/ko) until a release is being
published.

```bash
# In a checkout of this repository
$ export KO_DOCKER_REPO={prefix-for-image-reference} # e.g.: quay.io/vdemeest
$ ko apply -f config
2022/03/28 14:53:16 Using base gcr.io/distroless/static:nonroot@sha256:2556293984c5738fc75208cce52cf0a4762c709cf38e4bf8def65a61992da0ad for github.com/openshift-pipelines/tekton-task-group/cmd/controller
# […]
customresourcedefinition.apiextensions.k8s.io/taskgroups.custom.tekton.dev configured
deployment.apps/tekton-taskgroup-controller configured
```

This will build and install the `TaskGroup` controller on your
cluster, in the `tekton-pipelines` namespaces.

## Usage

**NOTE**:- You need to install [tektoncd/pipeline](https://github.com/tektoncd/pipeline/blob/main/docs/install.md) and also make sure to enable it's [alpha api](https://github.com/tektoncd/pipeline/blob/main/config/config-feature-flags.yaml#L72)

To execute a `TaskGroup` there is two ways :

- create a `Run` object that refers to `TaskGroup` — this is
  essentially like doing a `TaskRun`.
- create a `PipelineRun` object uses a Custom Task that refers to a
  `TaskGroup`.

In addition, there is examples (growing) in the
[`./examples`](./examples) repositories.

Both embedded spec *and* refering to a `TaskGroup` are support when
using `Run` or `PipelineRun`.

### `TaskGroup`

A `TaskGroup` is like a `Task` with an extra `uses` field

```yaml
apiVersion: custom.tekton.dev/v1alpha1
kind: TaskGroup
metadata:
  name: foo
spec:
  steps:
  - name: clone
    uses:
      taskRef:
        name: my-git-clone
  - name: build
    image: docker.io/library/golang:latest
    script: |
      ls -l /workspace; ls -l /workspace/foo; true
      cd /workspace/foo && go build -v ./...
```

### Using `Run`

To run a `foo` `TaskGroup` defined above, we can just define a `Run`
and refer to the `TaskGroup`. This is very similar to defining a
`TaskRun` that refers to a `Task`.

```yaml
apiVersion: tekton.dev/v1alpha1
kind: Run
metadata:
  name: my-run
spec:
  ref:
    apiVersion: custom.tekton.dev/v1alpha1
    kind: TaskGroup
    name: foo
```

It's also possible to use embedded spec, as following:

```yaml
apiVersion: tekton.dev/v1alpha1
kind: Run
metadata:
  name: my-embedded-run
spec:
  spec:
    apiVersion: custom.tekton.dev/v1alpha1
    kind: TaskGroup
    spec:
      steps:
      - name: clone
        uses:
          taskRef:
            name: my-git-clone
      - name: build
        image: docker.io/library/golang:latest
        script: |
          ls -l /workspace; ls -l /workspace/foo; true
          cd /workspace/foo && go build -v ./...
```

### Using `PipelineRun`

To use the `foo` `TaskGroup` defined above, we can refer to the custom
task and refer the `TaskGroup`.

```yaml
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: task-group-run-
spec:
  pipelineSpec:
    tasks:
      - name: task-group
        taskRef:
          apiVersion: custom.tekton.dev/v1alpha1
          kind: TaskGroup
          name: foo
        params:
        - name: git-url
          value: https://github.com/vdemeester/go-helloworld-app
      - name: bar
        runAfter: [task-group]
        taskRef:
          name: hello-world

```

### Parameters support

By default, `TaskGroup` would "augment" parameters that are coming
from all "injected" `Task` into our Task. There is two cases to take
into account though :

- **What happens if parameters are the same ?** Nothing, both refer to the same one
- **What happens if 2 parameters have different name but should have the same value ?** We need to allow the user to *map*
  a paramater from A into B.

It is thus possible to *bind* an injected task's parameter to a
`TaskGroup` parameter.

```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: git-clone
  # […]
spec:
  params:
  - name: url
    description: Repository URL to clone from.
    type: string
  - name: revision
    description: Revision to checkout. (branch, tag, sha, ref, etc...)
    type: string
    default: ""
  # […]
---
apiVersion: tekton.dev/v1beta1
kind: TaskGroup
metadata:
  name: foo
spec:
  params:
  - name: git_sha
    type: string
    description: represent the git sha to use
  steps:
  - uses:
    taskRef:
      name: git-clone
      parambindings:
      # this bind revision param from git-clone to this Task git_sha
      - name: revision
        param: git_sha
  - name: bar
    image: bash:latest
    script: |
      echo "baz"
  # […]
```

The following `yaml` is the "resolved" representation of the `Task` "foo".

```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: foo
spec:
  params:
  - name: git_sha
    type: string
    description: represent the git sha to use
  # Comining from `git-clone` Task
  - name: url
    description: #[…]
    # […]
  - name: refspec
    # […]
  # […]
  steps:
  - name: git-clone-clone
    image: "$(params.gitInitImage)"
    env:
    - name: HOME
      value: "$(params.userHome)"
    # […]
    - name: PARAM_REVISION
      value: $(params.git_sha) # we did bind revision to git_sha, meaning params.revision is replace by params.git_sha
    # […]
    script: |
      # […]
      /ko-app/git-init \
          -url="${PARAM_URL}" \
          -revision="${PARAM_REVISION}" \
          -refspec="${PARAM_REFSPEC}" \
          -path="${CHECKOUT_DIR}" \
          -sslVerify="${PARAM_SSL_VERIFY}" \
          -submodules="${PARAM_SUBMODULES}" \
          -depth="${PARAM_DEPTH}" \
          -sparseCheckoutDirectories="${PARAM_SPARSE_CHECKOUT_DIRECTORIES}"
      # […]
  - name: bar
    image: bash:latest
    script: |
      echo "baz"
```

### Workspaces support

Support for workspaces would work the same way as it does for `params`, with the same behavior for duplicated workspaces and
same workspaces with different names. It would look like the following:

```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: git-clone
  # […]
spec:
  workspaces:
  - name: output
    description: The git repo will be cloned onto the volume backing this Workspace.
  - name: ssh-directory
    optional: true
    # […]
  # […]
---
apiVersion: tekton.dev/v1beta1
kind: TaskGroup
metadata:
  name: foo
spec:
  workspaces:
  - name: sources
    type: string
    description: Sources to run something onto
  steps:
  - uses:
    taskRef:
      name: git-clone
      workspacebindings:
      # this bind revision workpace "output" from git-clone to this Task workspace "source"
      - name: output
        param: sources
  - name: bar
    image: bash:latest
    script: |
      ls -la $(workspaces.sources.path)
  # […]
```

## Limitations

There is a couple limitations in the current implementation (so far):

- Variable interpolation *outside* of the injected step by `uses` is
  not supported. This is only a problem in case you *bind* a parameter
  or a workspace.
