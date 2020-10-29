package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/adrianchifor/go-parallel"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	k8sNamespaces []string
	images        []string
)

func main() {
	getImages()
	sortImagesByPopularity()
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

	for _, ns := range k8sNamespaces {
		if ns == "*" {
			ns = ""
			log.Printf("Getting Pods in all namespaces ...")
		} else {
			log.Printf("Getting Pods in namespace '%s' ...", ns)
		}

		pods, err := k8s.CoreV1().Pods(ns).List(metav1.ListOptions{})
		if err != nil {
			log.Printf("Failed to get Pods in namespace '%s': %v", ns, err)
			continue
		}
		if len(pods.Items) == 0 {
			log.Printf("None found")
			continue
		}

		for _, pod := range pods.Items {
			log.Printf("Found '%s' with Docker images:", pod.Name)
			for _, container := range pod.Spec.InitContainers {
				log.Printf("- %s", container.Image)
				images = append(images, container.Image)
			}
			for _, container := range pod.Spec.Containers {
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
			k8sNamespaces = append(k8sNamespaces, strings.TrimSpace(ns))
		}
	} else {
		value, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			log.Fatalf("Failed to get current Kubernetes namespace: %v", err)
		}
		k8sNamespaces = append(k8sNamespaces, strings.TrimSpace(string(value)))
	}
}

func sortImagesByPopularity() {
	// Get image occurences
	imageCounts := make(map[string]int)
	for _, image := range images {
		imageCounts[image]++
	}

	// Remove duplicates
	encountered := make(map[string]struct{})
	result := []string{}

	for _, image := range images {
		if _, found := encountered[image]; !found {
			encountered[image] = struct{}{}
			result = append(result, image)
		}
	}

	// Sort result by highest occurence first
	sort.Slice(result, func(i, j int) bool {
		return imageCounts[result[j]] < imageCounts[result[i]]
	})

	// Limit images pulled if env set
	if value, ok := os.LookupEnv("LIMIT"); ok {
		limit, err := strconv.Atoi(value)
		if err != nil {
			log.Fatalf("Failed to convert LIMIT env to integer: %v", err)
		}
		if limit < len(result) {
			log.Printf("LIMIT env is set, only pulling top %d images", limit)
			result = result[:limit]
		}
	}

	images = result
}

func isRuntimeCrio() bool {
	sock, err := os.Stat("/var/run/crio/crio.sock")
	if os.IsNotExist(err) {
		return false
	}
	return !sock.IsDir()
}

func pullImages() {
	jobPool := parallel.SmallJobPool()
	defer jobPool.Close()

	binary := "/bin/docker"
	crio := isRuntimeCrio()
	if crio {
		binary = "/bin/crictl"
	}

	gcrAuthenticated := false

	for _, image := range images {
		image := image
		if !crio && strings.Contains(image, "gcr.io") && !gcrAuthenticated {
			out, err := exec.Command("/bin/docker-credential-gcr", "configure-docker").Output()
			if err != nil {
				log.Printf("Failed to authenticate with GCR: %v", err)
			} else {
				log.Println(string(out))
				gcrAuthenticated = true
			}
		}

		jobPool.AddJob(func() {
			out, err := exec.Command(binary, "pull", image).Output()
			if err != nil {
				log.Printf("Failed to pull Docker image '%s': %s : %v", image, string(out), err)
				return
			}

			log.Printf("Pulled '%s'", image)
		})
	}

	jobPool.Wait()
}
