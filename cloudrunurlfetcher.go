package cloudrunurlfetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"golang.org/x/oauth2/google"
)

// Minimal type to get only the interesting part in the answer
type cloudRunAPIUrlOnly struct {
	Status struct {
		URL string `json:"url"`
	} `json:"status"`
}

func getProjectAndRegion() (projectNumber string, region string, err error) {
	// Get the region from the metadata server. The project number is returned in the response
	resp, err := metadata.Get("/instance/region")
	if err != nil {
		return "", "", err
	}
	// response pattern is projects/<projectNumber>/regions/<region>
	r := strings.Split(resp, "/")
	projectNumber = r[1]
	region = r[3]
	return projectNumber, region, nil
}

func getCloudRunUrl(region string, projectNumber string, service string) (url string, err error) {
	ctx := context.Background()
	// To perform a call the Cloud Run API, the current service, through the service account, needs to be authenticated
	// The Google Auth default client add automatically the authorization header for that.
	client, err := google.DefaultClient(ctx)
	if err != nil {
		return "", fmt.Errorf("impossible to get the default client with error %+v", err)
	}

	// Build the request to the API
	cloudRunApi := fmt.Sprintf("https://%s-run.googleapis.com/apis/serving.knative.dev/v1/namespaces/%s/services/%s", region, projectNumber, service)
	// Perform the call
	resp, err := client.Get(cloudRunApi)
	if err != nil {
		return "", fmt.Errorf("error when calling the Cloud Run API %s with error %+v", cloudRunApi, err)
	}
	defer resp.Body.Close()

	// Read the body of the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("impossible to read the Cloud Run API call body with error %+v", err)
	}

	// Map the JSON body in a minimal struct. We need only the URL, the struct match only this part in the JSON.
	cloudRunResp := &cloudRunAPIUrlOnly{}
	json.Unmarshal(body, cloudRunResp)
	url = cloudRunResp.Status.URL
	return url, nil
}

// GetServiceURL returns the URL of the current Cloud Run service.
func GetServiceURL() (string, error) {
	// Get the projectNumber and the region of the current instance to call the right service
	projectNumber, region, err := getProjectAndRegion()
	if err != nil {
		return "", fmt.Errorf("impossible to get the projectNumber and region from the metadata server with error %+v", err)
	}

	// Get the service name from the environment variables
	// https://cloud.google.com/run/docs/container-contract#services-env-vars
	service := os.Getenv("K_SERVICE")
	if service == "" {
		return "", fmt.Errorf("impossible to get the Cloud Run service name from Environment Variable with error %+v", err)
	}

	// With the region, the projectNumber and the serviceName, it's possible to recover the Cloud Run service URL
	cloudRunUrl, err := getCloudRunUrl(region, projectNumber, service)
	if err != nil {
		return "", fmt.Errorf("impossible to get the Cloud Run service URL with error %+v", err)
	}
	return cloudRunUrl, nil
}
