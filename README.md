# multikubectl

A multi-cluster version of kubectl that allows you to run kubectl commands across multiple Kubernetes clusters simultaneously.

## Features

- **Multi-cluster support**: Query multiple Kubernetes clusters in a single command
- **Parallel execution**: Commands are executed concurrently across all clusters for optimal performance
- **CLUSTER column**: Adds a CLUSTER column to table outputs, showing which cluster each resource belongs to
- **Full kubectl compatibility**: Supports all kubectl commands, flags, and arguments
- **Flexible context selection**: Use all contexts, or specify a subset of clusters to query
- **Persistent configuration**: Save your preferred cluster selection to `~/.multikube/config`
- **Smart output merging**: Table outputs are merged with unified headers; non-table outputs (logs, describe) are displayed per-cluster

## Installation

### From Source

```bash
git clone https://github.com/multikubectl/multikubectl.git
cd multikubectl
go build -o multikubectl .
```

### Move to PATH (optional)

```bash
sudo mv multikubectl /usr/local/bin/
```

## Usage

```bash
multikubectl [flags] [kubectl command] [kubectl flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--kubeconfig` | Path to the kubeconfig file | `~/.kube/config` or `$KUBECONFIG` |
| `--contexts` | Comma-separated list of contexts to use (overrides config) | From config or all |
| `--all-contexts` | Use all available contexts (ignores config) | `false` |
| `--timeout` | Timeout for kubectl commands | `30s` |

### Examples

#### Get pods from all clusters

```bash
multikubectl get pods
```

Output:
```
CLUSTER     NAME                                   READY   STATUS        RESTARTS        AGE
cluster-a   volcano-admission-67496c9fdf-j7ch2     1/1     Terminating   27 (100d ago)   165d
cluster-a   volcano-admission-67496c9fdf-qd8vc     1/1     Running       0               3m29s
cluster-b   volcano-controllers-5b66bd9d47-7t4kl   1/1     Running       0               3m29s
cluster-b   volcano-controllers-5b66bd9d47-cw495   1/1     Terminating   25 (100d ago)   165d
cluster-c   volcano-scheduler-5bccfcf7d-kr66m      1/1     Running       0               3m28s
cluster-c   volcano-scheduler-5bccfcf7d-wqtgx      1/1     Terminating   26 (100d ago)   165d
```

#### Get pods from specific clusters

```bash
multikubectl --contexts=production,staging get pods -n kube-system
```

#### Get nodes with wide output

```bash
multikubectl get nodes -o wide
```

Output:
```
CLUSTER     NAME           STATUS   ROLES           AGE    VERSION   INTERNAL-IP    EXTERNAL-IP   OS-IMAGE             KERNEL-VERSION      CONTAINER-RUNTIME
cluster-a   node-1         Ready    control-plane   100d   v1.28.0   192.168.1.10   <none>        Ubuntu 22.04.3 LTS   5.15.0-91-generic   containerd://1.6.24
cluster-a   node-2         Ready    <none>          100d   v1.28.0   192.168.1.11   <none>        Ubuntu 22.04.3 LTS   5.15.0-91-generic   containerd://1.6.24
cluster-b   node-1         Ready    control-plane   50d    v1.29.0   10.0.0.10      <none>        Ubuntu 22.04.3 LTS   5.15.0-91-generic   containerd://1.7.0
```

#### Get deployments in a specific namespace

```bash
multikubectl get deployments -n default
```

#### View logs from a pod (non-table output)

```bash
multikubectl logs deployment/nginx
```

Output:
```
=== Cluster: cluster-a ===
10.0.0.1 - - [18/Jan/2026:10:00:00 +0000] "GET / HTTP/1.1" 200 612
10.0.0.2 - - [18/Jan/2026:10:00:01 +0000] "GET /health HTTP/1.1" 200 2

=== Cluster: cluster-b ===
10.0.1.1 - - [18/Jan/2026:10:00:00 +0000] "GET / HTTP/1.1" 200 612
```

#### Describe a resource

```bash
multikubectl describe service kubernetes
```

#### Use a custom kubeconfig

```bash
multikubectl --kubeconfig=/path/to/custom/config get pods
```

#### Set a custom timeout

```bash
multikubectl --timeout=60s get pods --all-namespaces
```

## How It Works

1. **Load kubeconfig**: Reads the kubeconfig file and extracts all available contexts
2. **Filter contexts**: If `--contexts` is specified, filters to only those contexts
3. **Parallel execution**: Executes the kubectl command against all selected contexts concurrently
4. **Output merging**:
   - For table outputs (get, top, etc.): Merges results and adds a CLUSTER column
   - For non-table outputs (logs, describe, etc.): Displays results grouped by cluster

## Supported Commands

multikubectl supports all kubectl commands. The output format handling differs based on command type:

### Table Output Commands
Commands that produce table output will have a CLUSTER column added:
- `get`
- `top`
- `api-resources`
- `api-versions`
- And more...

### Non-Table Output Commands
Commands that produce non-table output will be grouped by cluster:
- `logs`
- `describe`
- `explain`
- `exec`
- `attach`
- `port-forward`
- `proxy`
- `cp`

## Configuration

### Persistent Context Configuration

multikubectl supports saving your preferred cluster selection to `~/.multikube/config`. When this file exists, multikubectl will only query the configured contexts by default.

#### Config Commands

```bash
# Interactive multi-select (recommended)
multikubectl config select

# List all available contexts (shows which are configured with *)
multikubectl config list

# Set contexts to use (replaces existing config)
multikubectl config use production,staging

# Add context(s) to config
multikubectl config add development

# Remove context(s) from config
multikubectl config remove development

# Show current configuration
multikubectl config show

# Clear configuration (revert to using all contexts)
multikubectl config clear
```

#### Interactive Selection

The `config select` command provides an interactive multi-select interface:

```
$ multikubectl config select
? Select contexts to use (space to select, enter to confirm):
  [x] production
  [ ] staging
> [x] development
  [ ] testing
```

- Use **arrow keys** to navigate
- Press **space** to select/deselect
- Press **enter** to confirm
- Previously configured contexts are pre-selected

#### Example Workflow

```bash
# First, see all available contexts
$ multikubectl config list
Available contexts from kubeconfig:

  production (current)
  staging
  development
  testing

No multikube config found. Using all contexts.
Run 'multikubectl config use <contexts>' to configure specific contexts.

# Configure to only use production and staging
$ multikubectl config use production,staging
Configured contexts:
  - production
  - staging

Configuration saved to /home/user/.multikube/config

# Now commands will only query these clusters
$ multikubectl get pods
CLUSTER      NAME                    READY   STATUS    RESTARTS   AGE
production   nginx-abc123            1/1     Running   0          10d
staging      nginx-xyz789            1/1     Running   0          5d

# Override config temporarily with --contexts flag
$ multikubectl --contexts=development get pods

# Or query all clusters with --all-contexts
$ multikubectl --all-contexts get pods
```

#### Context Resolution Priority

1. `--contexts` flag (highest priority)
2. `--all-contexts` flag
3. `~/.multikube/config` file
4. All contexts from kubeconfig (default)

### Environment Variables

- `KUBECONFIG`: Path to the kubeconfig file (can be overridden with `--kubeconfig`)

### Kubeconfig

multikubectl uses the standard Kubernetes kubeconfig format. Each context in your kubeconfig represents a cluster that can be queried.

Example kubeconfig with multiple contexts:

```yaml
apiVersion: v1
kind: Config
current-context: production
contexts:
- name: production
  context:
    cluster: prod-cluster
    user: prod-user
- name: staging
  context:
    cluster: staging-cluster
    user: staging-user
- name: development
  context:
    cluster: dev-cluster
    user: dev-user
clusters:
- name: prod-cluster
  cluster:
    server: https://prod.example.com:6443
- name: staging-cluster
  cluster:
    server: https://staging.example.com:6443
- name: dev-cluster
  cluster:
    server: https://dev.example.com:6443
users:
- name: prod-user
  user:
    token: <token>
- name: staging-user
  user:
    token: <token>
- name: dev-user
  user:
    token: <token>
```

## Error Handling

When a command fails on one or more clusters, multikubectl will:
1. Display the error message for the failed cluster(s)
2. Continue showing results from successful clusters
3. Exit with a non-zero status code

Example:
```
CLUSTER     NAME                    READY   STATUS    RESTARTS   AGE
cluster-a   nginx-7c5ddbdf54-abc    1/1     Running   0          10d
cluster-b   nginx-7c5ddbdf54-xyz    1/1     Running   0          5d
# Error from cluster cluster-c: The connection to the server was refused
```

## Requirements

- Go 1.21+ (for building from source)
- kubectl installed and available in PATH
- Valid kubeconfig with one or more contexts

## License

MIT License
