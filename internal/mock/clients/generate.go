// Package clients define configuration for generating mocks of existing clients.
//
//go:generate go run github.com/golang/mock/mockgen -source=../../clients/k8s/k8snodepool/nodepool.go -destination=k8s/k8snodepool/nodepool.go -package k8snodepool
//go:generate go run github.com/golang/mock/mockgen -source=../../clients/k8s/k8scluster/cluster.go -destination=k8s/k8scluster/cluster.go -package k8scluster
//go:generate go run github.com/golang/mock/mockgen -source=../../clients/compute/datacenter/datacenter.go -destination=compute/datacenter/datacenter.go -package datacenter
//go:generate go run github.com/golang/mock/mockgen -source=../../clients/compute/ipblock/ipblock.go -destination=compute/ipblock/ipblock.go -package ipblock
//go:generate go run github.com/golang/mock/mockgen -source=../../clients/compute/user/user.go -destination=compute/user/user.go -package user
package clients
