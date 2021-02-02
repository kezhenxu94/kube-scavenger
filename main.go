package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"gopkg.in/matryer/try.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

var port = flag.Int("p", 8080, "Port to bind at")

func main() {
	flag.Parse()

	log.Println("Pinging API Server...")

	var clients *kubernetes.Clientset

	if config, err := rest.InClusterConfig(); err != nil {
		panic(err.Error())
	} else if clients, err = kubernetes.NewForConfig(config); err != nil {
		panic(err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := clients.Discovery().RESTClient().Get().AbsPath("/healthz").DoRaw(ctx); err != nil {
		panic(err.Error())
	}

	log.Println("Connect to API server successfully!")

	selectors := make(map[string][]string)

	firstConnected := make(chan bool, 1)
	var wg sync.WaitGroup

	go processRequests(selectors, firstConnected, &wg)

	select {
	case <-time.After(1 * time.Minute):
		panic("Timed out waiting for the first connection")
	case <-firstConnected:
	}
	log.Println("Received the first connection")

	wg.Wait()
	log.Println("Timed out waiting for re-connection")

	prune(clients, selectors)
}

func processRequests(selectors map[string][]string, firstConnected chan<- bool, wg *sync.WaitGroup) {
	var once sync.Once

	log.Printf("Starting on port %d...", *port)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))

	if err != nil {
		panic(err)
	}
	log.Println("Started!")

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		log.Printf("New client connected: %s\n", conn.RemoteAddr().String())
		go func() {
			wg.Add(1)
			defer wg.Done()
			once.Do(func() {
				firstConnected <- true
			})
			reader := bufio.NewReader(conn)
			for {
				message, err := reader.ReadString('\n')

				message = strings.TrimSpace(message)

				if len(message) > 0 {
					query, err := url.ParseQuery(message)

					if err != nil {
						log.Println(err)
						continue
					}

					for key, val := range query {
						for _, v := range val {
							log.Printf("Adding %s=%s\n", key, v)
						}
						selectors[key] = val
					}

					_, _ = conn.Write([]byte("ACK\n"))
				}

				if err != nil {
					log.Println(err)
					break
				}
			}
			log.Printf("Client disconnected: %s\n", conn.RemoteAddr().String())
			_ = conn.Close()

			time.Sleep(10 * time.Second)
		}()
	}
}

func prune(cli *kubernetes.Clientset, selectors map[string][]string) {
	deletedPods := make(map[string]bool)
	deletedDeployments := make(map[string]bool)
	deletedServices := make(map[string]bool)
	deletedNamespaces := make(map[string]bool)

	for key, vals := range selectors {
		for _, val := range vals {
			selector := fmt.Sprintf("%s=%s", key, val)

			log.Printf("Deleting %s=%s\n", key, val)

			_ = try.Do(func(attempt int) (bool, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				namespaces, err := cli.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				if err != nil {
					log.Printf("Listing namespaces has failed, retrying(%d/%d). The error was: %v", attempt, 10, err)
					time.Sleep(1 * time.Second)
					return attempt < 10, err
				}
				for _, ns := range namespaces.Items {
					for k, v := range ns.GetLabels() {
						if k == key && v == val {
							deleteNamespace(cli, ns, selector, deletedNamespaces)
						}
					}
					deleteServices(cli, ns, selector, deletedServices)
					deleteDeployments(cli, ns, selector, deletedServices)
					deletePods(cli, ns, selector, deletedServices)
				}
				return false, nil
			})
		}
	}

	log.Printf("Removed %d container(s), %d network(s), %d volume(s) %d image(s)", len(deletedPods), len(deletedDeployments), len(deletedServices), len(deletedNamespaces))
}

func deleteNamespace(cli *kubernetes.Clientset, ns corev1.Namespace, _ string, deletedNamespaces map[string]bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := cli.CoreV1().Namespaces().Delete(ctx, ns.Name, metav1.DeleteOptions{}); err != nil {
		log.Println(err)
		deletedNamespaces[ns.Name] = false
	}
	deletedNamespaces[ns.Name] = true
}

func deleteServices(cli *kubernetes.Clientset, ns corev1.Namespace, selector string, deletedServices map[string]bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	services, err := cli.CoreV1().Services(ns.Name).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Println(err)
	}
	for _, svc := range services.Items {
		if err := cli.CoreV1().Services(ns.Name).Delete(ctx, svc.Name, metav1.DeleteOptions{}); err != nil {
			log.Println(err)
		} else {
			deletedServices[svc.Name] = true
		}
	}
}

func deleteDeployments(cli *kubernetes.Clientset, ns corev1.Namespace, selector string, deletedDeployments map[string]bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployments, err := cli.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Println(err)
		return
	}
	for _, svc := range deployments.Items {
		if err := cli.CoreV1().Services(ns.Name).Delete(ctx, svc.Name, metav1.DeleteOptions{}); err != nil {
			log.Println(err)
		} else {
			deletedDeployments[svc.Name] = true
		}
	}
}

func deletePods(cli *kubernetes.Clientset, ns corev1.Namespace, selector string, deletedPods map[string]bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pods, err := cli.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Println(err)
	}
	for _, pod := range pods.Items {
		if err := cli.CoreV1().Pods(ns.Name).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
			log.Println(err)
		} else {
			deletedPods[pod.Name] = true
		}
	}
}
