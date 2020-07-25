# This file is an example of using starlark + skycfg for generating Kubernetes configs
# for a simple Redis deployment. See https://github.com/stripe/skycfg for more details.
kube = struct(
  appsv1 = proto.package("k8s.io.api.apps.v1"),
  corev1 = proto.package("k8s.io.api.core.v1"),
  metav1 = proto.package("k8s.io.apimachinery.pkg.apis.meta.v1"),
)

def main(ctx):
    redis_container = kube.corev1.Container(
        name='redis',
        image='redis:%s' % ctx.vars['parameters']['redis']['imageTag'],
        ports=[
            kube.corev1.ContainerPort(containerPort=6379)
        ],
    )

    redis_deployment = kube.appsv1.Deployment(
        metadata=kube.metav1.ObjectMeta(
            name='redis',
            namespace='apps',
        ),
        spec=kube.appsv1.DeploymentSpec(
            replicas=ctx.vars['parameters']['redis']['replicas'],
            selector=kube.metav1.LabelSelector(
                matchLabels={'app': 'redis'},
            ),
            template=kube.corev1.PodTemplateSpec(
                metadata=kube.metav1.ObjectMeta(
                    name='redis',
                    labels={'app': 'redis'},
                ),
                spec=kube.corev1.PodSpec(
                    containers=[redis_container],
                ),
            ),
        ),
    )

    return [
        redis_deployment,
    ]
