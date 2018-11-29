package nodefeaturediscovery


var nfdserviceaccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: node-feature-discovery
  namespace: openshift-cluster-nfd-operator
`

var nfdclusterrole = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: node-feature-discovery
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - nodes
  verbs:
  - get
  - patch
  - update
`

var nfdclusterrolebinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: node-feature-discovery
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: node-feature-discovery
subjects:
- kind: ServiceAccount
  name: node-feature-discovery
  namespace: node-feature-discovery
`

var nfdscc = `
allowHostDirVolumePlugin: true
allowHostIPC: false
allowHostNetwork: true
allowHostPID: false
allowHostPorts: false
allowPrivilegeEscalation: true
allowPrivilegedContainer: false
allowedCapabilities: null
apiVersion: security.openshift.io/v1
defaultAddCapabilities: null
fsGroup:
  type: RunAsAny
groups: []
kind: SecurityContextConstraints
metadata:
  annotations:
    kubernetes.io/description: 'hostmount-anyuid provides all the features of the
      restricted SCC but allows host mounts and any UID by a pod.  This is primarily
      used by the persistent volume recycler. WARNING: this SCC allows host file system
      access as any UID, including UID 0.  Grant with caution.'
  creationTimestamp: 2018-10-23T09:00:53Z
  name: node-feature-discovery
  resourceVersion: "156705"
  selfLink: /apis/security.openshift.io/v1/securitycontextconstraints/node-feature-discovery
  uid: 253d2129-d6a2-11e8-a7ea-06f9c2879cd2
priority: null
readOnlyRootFilesystem: false
requiredDropCapabilities:
- MKNOD
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: MustRunAs
supplementalGroups:
  type: RunAsAny
users:
- system:serviceaccount:openshift-infra:pv-recycler-controller
- system:serviceaccount:kube-service-catalog:service-catalog-apiserver
- system:serviceaccount:node-feature-discovery:node-feature-discovery
volumes:
- configMap
- downwardAPI
- emptyDir
- hostPath
- nfs
- persistentVolumeClaim
- projected
- secret
`
var nfddaemonset = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    app: node-feature-discovery
  name: node-feature-discovery
spec:
  selector:
    matchLabels:
      app: node-feature-discovery
  template:
    metadata:
      labels:
        app: node-feature-discovery
    spec:
      hostNetwork: true
      serviceAccount: node-feature-discovery
      containers:
        - image: quay.io/zvonkok/node-feature-discovery:v0.3.0-10-g86947fc-dirty
          name: node-feature-discovery
          command: ["/usr/bin/node-feature-discovery", "--source=pci"]
          args:
            - "--sleep-interval=60s"
          volumeMounts:
            - name: host-sys
              mountPath: "/host-sys"
      volumes:
        - name: host-sys
          hostPath:
            path: "/sys"
`


var nfddaemonset2 = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    app: node-feature-discovery
  name: node-feature-discovery
spec:
  selector:
    matchLabels:
      app: node-feature-discovery
  template:
    metadata:
      labels:
        app: node-feature-discovery
    spec:
      hostNetwork: true
      serviceAccount: node-feature-discovery
      containers:
        - env:3
          - name: NODE_NAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
          image: quay.io/zvonkok/node-feature-discovery:v0.3.0-10-g86947fc-dirty
          name: node-feature-discovery
          command: ["/usr/bin/node-feature-discovery", "--source=pci"]
          args:
            - "--sleep-interval=60s"
          volumeMounts:
            - name: host-sys
              mountPath: "/host-sys"
      volumes:
        - name: host-sys
          hostPath:
            path: "/sys"
`
