load("deploy.star", "deployment")

def main(ctx):
  return [
    deployment("nginx"),
    deployment("haproxy")
  ] + util.fromYaml("""
apiVersion: v1
kind: Service
metadata:
  name: kafka
  namespace: centrifuge
  labels:
    app: kafka
  annotations:
    service.alpha.kubernetes.io/tolerate-unready-endpoints: "true"
    external-dns.alpha.kubernetes.io/hostname: "kafka.centrifuge-destinations"
spec:
  ports:
  - name: broker
    port: 9092
    targetPort: 44445
  clusterIP: None
  selector:
    app: kafka
    """,
    )
