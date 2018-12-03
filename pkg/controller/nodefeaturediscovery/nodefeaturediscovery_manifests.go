package nodefeaturediscovery

var nfdserviceaccount = []byte(`
apiVersion: v1
kind: ServiceAccount
metadata:
  name: node-feature-discovery
  namespace: openshift-cluster-nfd-operator
`)

var nfdclusterrole = []byte(`
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
- apiGroups:
  - security.openshift.io
  resources:
  - securitycontextconstraints
  verbs:
  - use
  resourceNames:
  - hostnetwork
`)

var nfdclusterrolebinding = []byte(`
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
  namespace: openshift-cluster-nfd-operator
userNames:
- system:serviceaccount:openshift-cluster-nfd-operator:node-feature-discovery
`)

var nfdsecuritycontextconstraint = []byte(`
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
- system:serviceaccount:openshift-cluster-nfd-operator:node-feature-discovery
volumes:
- configMap
- downwardAPI
- emptyDir
- hostPath
- nfs
- persistentVolumeClaim
- projected
- secret
`)


var nfdconfigmap = []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: node-feature-discovery
  namespace: openshift-cluster-nfd-operator
data:
  node-feature-discovery-conf: |
    #sources:
    #  pci:
    #    deviceClassWhitelist:
    #      - "0200"
    #      - "03"
    #      - "12"
    #    deviceLabelFields:
    #      - "class"
    #      - "vendor"
    #      - "device"
    #      - "subsystem_vendor"
    #      - "subsystem_device"
`)





var nfddaemonset = []byte(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    app: node-feature-discovery
  name: node-feature-discovery
  namespace: openshift-cluster-nfd-operator
spec:
  selector:
    matchLabels:
      app: node-feature-discovery
  template:
    metadata:
      labels:
        app: node-feature-discovery
    spec:
#      hostNetwork: true
      serviceAccount: node-feature-discovery
      containers:
        - env:
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
          securityContext:
            privileged: true
      volumes:
        - name: host-sys
          hostPath:
            path: "/sys"
`)
