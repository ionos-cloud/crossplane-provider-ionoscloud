//go:generate go run github.com/golang/mock/mockgen -source=../../clients/compute/datacenter/datacenter.go -destination=compute/datacenter/datacenter.go -package datacenter
//go:generate go run github.com/golang/mock/mockgen -source=../../clients/compute/ipblock/ipblock.go -destination=compute/ipblock/ipblock.go -package ipblock
//go:generate go run github.com/golang/mock/mockgen -source=../../clients/compute/user/user.go -destination=compute/user/user.go -package user

//go:generate go run github.com/golang/mock/mockgen -source=../../clients/k8s/k8snodepool/nodepool.go -destination=k8s/k8snodepool/nodepool.go -package k8snodepool
//go:generate go run github.com/golang/mock/mockgen -source=../../clients/k8s/k8scluster/cluster.go -destination=k8s/k8scluster/cluster.go -package k8scluster
package clients
