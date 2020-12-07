kube = struct(
  appsv1 = proto.package("k8s.io.api.apps.v1"),
  corev1 = proto.package("k8s.io.api.core.v1"),
  metav1 = proto.package("k8s.io.apimachinery.pkg.apis.meta.v1"),
)

def container(name):
  return kube.corev1.Container(
    name = name,
    image = name + ":latest",
    ports = [
      kube.corev1.ContainerPort(containerPort = 80)
    ],
    resources = kube.corev1.ResourceRequirements(
      requests = {
        "cpu": util.quantity("300m"),
        "memory": util.quantity("5G"),
      },
      limits = {
        "cpu": util.quantity("300m"),
        "memory": util.quantity("2G"),
      },
    ),
  )

def deployment(name):
  d = kube.appsv1.Deployment()
  d.metadata.name = name

  spec = d.spec
  spec.selector = kube.metav1.LabelSelector(
    matchLabels = {"app": name},
  )
  spec.replicas = 1

  tmpl = spec.template
  tmpl.metadata.labels = {"app": name}
  tmpl.spec.volumes = [
    kube.corev1.Volume(
      name = "test-volume",
      volumeSource = kube.corev1.VolumeSource(
        hostPath = kube.corev1.HostPathVolumeSource(
          path = "/vol/path",
        ),
      ),
    ),
  ]

  tmpl.spec.containers = [
    container(name),
  ]

  return d
