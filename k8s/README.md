
kubectl apply -f bench-testcluster.yaml
kubectl get pods

kubectl logs chuckd-bench-testcluster-quorum-0 | l
kubectl logs -f chuckd-bench-testcluster-quorum-0

kubectl exec -it chuckd-bench-testcluster-quorum-0 -- ps aux
kubectl exec -it chuckd-bench-testcluster-quorum-0 -- kill -2 1

kubectl delete statefulset --all

kubectl get statefulsets
kubectl delete statefulset NAME_OF_STATEFULSET
