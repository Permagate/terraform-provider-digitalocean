package digitalocean

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	yaml "gopkg.in/yaml.v2"
)

func resourceDigitalOceanKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		Create:        resourceDigitalOceanKubernetesClusterCreate,
		Read:          resourceDigitalOceanKubernetesClusterRead,
		Update:        resourceDigitalOceanKubernetesClusterUpdate,
		Delete:        resourceDigitalOceanKubernetesClusterDelete,
		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(val interface{}) string {
					// DO API V2 region slug is always lowercase
					return strings.ToLower(val.(string))
				},
				ValidateFunc: validation.NoZeroValues,
			},

			"version": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"cluster_subnet": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"service_subnet": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv4_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),

			"node_pool": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: nodePoolSchema(),
				},
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"kube_config": kubernetesConfigSchema(),
		},
	}
}

func kubernetesConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type:      schema.TypeList,
		Computed:  true,
		Sensitive: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"raw_config": {
					Type:     schema.TypeString,
					Computed: true,
				},

				"host": {
					Type:     schema.TypeString,
					Computed: true,
				},

				"cluster_ca_certificate": {
					Type:     schema.TypeString,
					Computed: true,
				},

				"client_key": {
					Type:     schema.TypeString,
					Computed: true,
				},

				"client_certificate": {
					Type:     schema.TypeString,
					Computed: true,
				},

				"token": {
					Type:     schema.TypeString,
					Computed: true,
				},

				"expires_at": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	}
}

func resourceDigitalOceanKubernetesClusterCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*CombinedConfig).godoClient()

	pools := expandNodePools(d.Get("node_pool").([]interface{}))
	poolCreateRequests := make([]*godo.KubernetesNodePoolCreateRequest, len(pools))
	for i, pool := range pools {
		tags := append(pool.Tags, digitaloceanKubernetesDefaultNodePoolTag)
		poolCreateRequests[i] = &godo.KubernetesNodePoolCreateRequest{
			Name:  pool.Name,
			Size:  pool.Size,
			Tags:  tags,
			Count: pool.Count,
		}
	}

	opts := &godo.KubernetesClusterCreateRequest{
		Name:        d.Get("name").(string),
		RegionSlug:  d.Get("region").(string),
		VersionSlug: d.Get("version").(string),
		Tags:        expandTags(d.Get("tags").(*schema.Set).List()),
		NodePools:   poolCreateRequests,
	}

	cluster, _, err := client.Kubernetes.Create(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("Error creating Kubernetes cluster: %s", err)
	}

	// wait for completion
	cluster, err = waitForKubernetesClusterCreate(client, cluster.ID)
	if err != nil {
		return fmt.Errorf("Error creating Kubernetes cluster: %s", err)
	}

	// set the cluster id
	d.SetId(cluster.ID)

	return resourceDigitalOceanKubernetesClusterRead(d, meta)
}

func resourceDigitalOceanKubernetesClusterRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*CombinedConfig).godoClient()

	cluster, resp, err := client.Kubernetes.Get(context.Background(), d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Kubernetes cluster: %s", err)
	}

	return digitaloceanKubernetesClusterRead(client, cluster, d)
}

func digitaloceanKubernetesClusterRead(client *godo.Client, cluster *godo.KubernetesCluster, d *schema.ResourceData) error {
	d.Set("name", cluster.Name)
	d.Set("region", cluster.RegionSlug)
	d.Set("version", cluster.VersionSlug)
	d.Set("cluster_subnet", cluster.ClusterSubnet)
	d.Set("service_subnet", cluster.ServiceSubnet)
	d.Set("ipv4_address", cluster.IPv4)
	d.Set("endpoint", cluster.Endpoint)
	d.Set("tags", flattenTags(filterTags(cluster.Tags)))
	d.Set("status", cluster.Status.State)
	d.Set("created_at", cluster.CreatedAt.UTC().String())
	d.Set("updated_at", cluster.UpdatedAt.UTC().String())

	// find the default node pool from all the pools in the cluster
	// the default node pool has a custom tag k8s:default-node-pool
	for _, p := range cluster.NodePools {
		for _, t := range p.Tags {
			if t == digitaloceanKubernetesDefaultNodePoolTag {
				if err := d.Set("node_pool", flattenNodePool(p, cluster.Tags...)); err != nil {
					log.Printf("[DEBUG] Error setting node pool attributes: %s %#v", err, cluster.NodePools)
				}
			}
		}
	}

	// fetch cluster credentials and update the resource if the credentials are expired.
	var creds map[string]interface{}
	if d.Get("kube_config") != nil && len(d.Get("kube_config").([]interface{})) > 0 {
		creds = d.Get("kube_config").([]interface{})[0].(map[string]interface{})
	}
	var expiresAt time.Time
	if creds["expires_at"] != nil && creds["expires_at"].(string) != "" {
		var err error
		expiresAt, err = time.Parse(time.RFC3339, creds["expires_at"].(string))
		if err != nil {
			return fmt.Errorf("Unable to parse Kubernetes credentials expiry: %s", err)
		}
	}
	if expiresAt.IsZero() || expiresAt.Before(time.Now()) {
		creds, resp, err := client.Kubernetes.GetCredentials(context.Background(), cluster.ID, &godo.KubernetesClusterCredentialsGetRequest{})
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				return fmt.Errorf("Unable to fetch Kubernetes credentials: %s", err)
			}
		}
		d.Set("kube_config", flattenCredentials(cluster.Name, cluster.RegionSlug, creds))
	}

	return nil
}

func resourceDigitalOceanKubernetesClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*CombinedConfig).godoClient()

	// Figure out the changes and then call the appropriate API methods
	if d.HasChange("name") || d.HasChange("tags") {

		opts := &godo.KubernetesClusterUpdateRequest{
			Name: d.Get("name").(string),
			Tags: expandTags(d.Get("tags").(*schema.Set).List()),
		}

		_, resp, err := client.Kubernetes.Update(context.Background(), d.Id(), opts)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				d.SetId("")
				return nil
			}

			return fmt.Errorf("Unable to update cluster: %s", err)
		}
	}

	// Update the node pool if necessary
	if !d.HasChange("node_pool") {
		return resourceDigitalOceanKubernetesClusterRead(d, meta)
	}

	old, new := d.GetChange("node_pool")
	oldPool := old.([]interface{})[0].(map[string]interface{})
	newPool := new.([]interface{})[0].(map[string]interface{})

	// update the existing default pool
	_, err := digitaloceanKubernetesNodePoolUpdate(client, newPool, d.Id(), oldPool["id"].(string), digitaloceanKubernetesDefaultNodePoolTag)
	if err != nil {
		return err
	}

	return resourceDigitalOceanKubernetesClusterRead(d, meta)
}

func resourceDigitalOceanKubernetesClusterDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*CombinedConfig).godoClient()

	resp, err := client.Kubernetes.Delete(context.Background(), d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Unable to delete cluster: %s", err)
	}

	d.SetId("")

	return nil
}

func waitForKubernetesClusterCreate(client *godo.Client, id string) (*godo.KubernetesCluster, error) {
	ticker := time.NewTicker(10 * time.Second)
	timeout := 120
	n := 0

	for range ticker.C {
		cluster, _, err := client.Kubernetes.Get(context.Background(), id)
		if err != nil {
			ticker.Stop()
			return nil, fmt.Errorf("Error trying to read cluster state: %s", err)
		}

		if cluster.Status.State == "running" {
			ticker.Stop()
			return cluster, nil
		}

		if n > timeout {
			ticker.Stop()
			break
		}

		n++
	}

	return nil, fmt.Errorf("Timeout waiting to create cluster")
}

type kubernetesConfig struct {
	APIVersion     string                    `yaml:"apiVersion"`
	Kind           string                    `yaml:"kind"`
	Clusters       []kubernetesConfigCluster `yaml:"clusters"`
	Contexts       []kubernetesConfigContext `yaml:"contexts"`
	CurrentContext string                    `yaml:"current-context"`
	Users          []kubernetesConfigUser    `yaml:"users"`
}

type kubernetesConfigCluster struct {
	Cluster kubernetesConfigClusterData `yaml:"cluster"`
	Name    string                      `yaml:"name"`
}
type kubernetesConfigClusterData struct {
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	Server                   string `yaml:"server"`
}

type kubernetesConfigContext struct {
	Context kubernetesConfigContextData `yaml:"context"`
	Name    string                      `yaml:"name"`
}
type kubernetesConfigContextData struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

type kubernetesConfigUser struct {
	Name string                   `yaml:"name"`
	User kubernetesConfigUserData `yaml:"user"`
}

type kubernetesConfigUserData struct {
	ClientKeyData         string `yaml:"client-key-data,omitempty"`
	ClientCertificateData string `yaml:"client-certificate-data,omitempty"`
	Token                 string `yaml:"token"`
}

func flattenCredentials(name string, region string, creds *godo.KubernetesClusterCredentials) []interface{} {
	raw := map[string]interface{}{
		"cluster_ca_certificate": base64.StdEncoding.EncodeToString(creds.CertificateAuthorityData),
		"host":                   creds.Server,
		"token":                  creds.Token,
		"expires_at":             creds.ExpiresAt.Format(time.RFC3339),
	}

	if creds.ClientKeyData != nil {
		raw["client_key"] = string(creds.ClientKeyData)
	}

	if creds.ClientCertificateData != nil {
		raw["client_certificate"] = string(creds.ClientCertificateData)
	}

	kubeconfigYAML, err := renderKubeconfig(name, region, creds)
	if err != nil {
		log.Printf("[DEBUG] error marshalling config: %s", err)
		return nil
	}
	raw["raw_config"] = string(kubeconfigYAML)

	return []interface{}{raw}
}

func renderKubeconfig(name string, region string, creds *godo.KubernetesClusterCredentials) ([]byte, error) {
	clusterName := fmt.Sprintf("do-%s-%s", region, name)
	userName := fmt.Sprintf("do-%s-%s-admin", region, name)
	config := kubernetesConfig{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []kubernetesConfigCluster{{
			Name: clusterName,
			Cluster: kubernetesConfigClusterData{
				CertificateAuthorityData: base64.StdEncoding.EncodeToString(creds.CertificateAuthorityData),
				Server:                   creds.Server,
			},
		}},
		Contexts: []kubernetesConfigContext{{
			Context: kubernetesConfigContextData{
				Cluster: clusterName,
				User:    userName,
			},
			Name: clusterName,
		}},
		CurrentContext: clusterName,
		Users: []kubernetesConfigUser{{
			Name: userName,
			User: kubernetesConfigUserData{
				Token: creds.Token,
			},
		}},
	}
	if creds.ClientKeyData != nil {
		config.Users[0].User.ClientKeyData = base64.StdEncoding.EncodeToString(creds.ClientKeyData)
	}
	if creds.ClientCertificateData != nil {
		config.Users[0].User.ClientCertificateData = base64.StdEncoding.EncodeToString(creds.ClientCertificateData)
	}
	return yaml.Marshal(config)
}

// we need to filter tags to remove any automatically added to avoid state problems,
// these are tags starting with "k8s:" or named "k8s"
func filterTags(tags []string) []string {
	filteredTags := make([]string, 0)
	for _, t := range tags {
		if !strings.HasPrefix(t, "k8s:") &&
			!strings.HasPrefix(t, "terraform:") &&
			t != "k8s" {
			filteredTags = append(filteredTags, t)
		}
	}

	return filteredTags
}
