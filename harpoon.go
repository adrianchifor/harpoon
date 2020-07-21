package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/adrianchifor/go-parallel"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	k8s_namespaces []string
	images         []string
)

func main() {
	getImages()
	removeDuplicateImages()
	pullImages()
}

func getImages() {
	if imagesConfigExists() {
		images = getImagesFromConfig()
		return
	}

	initNamespaces()

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to get Kubernetes in-cluster config: %v", err)
	}
	k8s, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	for _, ns := range k8s_namespaces {
		if ns == "*" {
			ns = ""
			log.Printf("Getting Deployments in all namespaces ...")
		} else {
			log.Printf("Getting Deployments in namespace '%s' ...", ns)
		}

		deploys, err := k8s.AppsV1().Deployments(ns).List(metav1.ListOptions{})
		if err != nil {
			log.Printf("Failed to get Deployments in namespace '%s': %v", ns, err)
			continue
		}
		if len(deploys.Items) == 0 {
			log.Printf("None found")
			continue
		}

		for _, deploy := range deploys.Items {
			log.Printf("Found '%s' with Docker images:", deploy.Name)
			for _, container := range deploy.Spec.Template.Spec.InitContainers {
				log.Printf("- %s", container.Image)
				images = append(images, container.Image)
			}
			for _, container := range deploy.Spec.Template.Spec.Containers {
				log.Printf("- %s", container.Image)
				images = append(images, container.Image)
			}
		}
	}
}

func imagesConfigExists() bool {
	file, err := os.Stat("/config/images")
	if os.IsNotExist(err) {
		return false
	}
	return !file.IsDir()
}

func getImagesFromConfig() []string {
	log.Printf("Getting Docker images from /config/images ...")
	file, err := os.Open("/config/images")
	if err != nil {
		log.Fatalf("Failed opening /config/images: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	return lines
}

func initNamespaces() {
	if value, ok := os.LookupEnv("NAMESPACES"); ok {
		namespaces := strings.Split(value, ",")
		if len(namespaces) == 0 {
			log.Fatalf("Specify one or more namespaces (comma-separated) in 'NAMESPACES' env var, or '*' for all namespaces")
		}
		for _, ns := range namespaces {
			k8s_namespaces = append(k8s_namespaces, strings.TrimSpace(ns))
		}
	} else {
		value, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			log.Fatalf("Failed to get current Kubernetes namespace: %v", err)
		}
		k8s_namespaces = append(k8s_namespaces, strings.TrimSpace(string(value)))
	}
}

func removeDuplicateImages() {
	encountered := make(map[string]struct{})
	result := []string{}

	for _, image := range images {
		if _, found := encountered[image]; !found {
			encountered[image] = struct{}{}
			result = append(result, image)
		}
	}

	images = result
}

func pullImages() {
	jobPool := parallel.SmallJobPool()
	defer jobPool.Close()

	gcrAuthenticated := false

	for _, image := range images {
		image := image
		if strings.Contains(image, "gcr.io") && !gcrAuthenticated {
			out, err := exec.Command("/bin/docker-credential-gcr", "configure-docker").Output()
			if err != nil {
				log.Printf("Failed to authenticate with GCR: %v", err)
			} else {
				log.Println(string(out))
				gcrAuthenticated = true
			}
		}

		jobPool.AddJob(func() {
			out, err := exec.Command("/bin/docker", "pull", image).Output()
			if err != nil {
				log.Printf("Failed to pull Docker image '%s': %s", image, string(out))
				return
			}

			log.Printf("Pulled '%s'", image)
		})
	}

	jobPool.Wait()
}
