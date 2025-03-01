package digitalocean

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccDigitalOceanKubernetesCluster_Basic(t *testing.T) {
	t.Parallel()
	rName := randomTestName()
	var k8s godo.KubernetesCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanKubernetesClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDigitalOceanKubernetesConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanKubernetesClusterExists("digitalocean_kubernetes_cluster.foobar", &k8s),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "name", rName),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "region", "lon1"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "version", "1.15.4-do.0"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "ipv4_address"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "cluster_subnet"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "service_subnet"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "endpoint"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "tags.#", "3"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "tags.2356372769", "foo"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "tags.1996459178", "bar"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "status"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "created_at"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "updated_at"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.#", "1"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.size", "s-1vcpu-2gb"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.node_count", "1"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.tags.#", "2"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.tags.2053932785", "one"), // Currently tags are being copied from parent this will fail
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.tags.298486374", "two"),  // requires API update
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.nodes.#", "1"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "node_pool.0.nodes.0.name"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "node_pool.0.nodes.0.status"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "node_pool.0.nodes.0.created_at"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "node_pool.0.nodes.0.updated_at"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "kube_config.0.raw_config"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "kube_config.0.cluster_ca_certificate"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "kube_config.0.host"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "kube_config.0.token"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "kube_config.0.expires_at"),
				),
			},
		},
	})
}

func TestAccDigitalOceanKubernetesCluster_UpdateCluster(t *testing.T) {
	t.Parallel()
	rName := randomTestName()
	var k8s godo.KubernetesCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanKubernetesClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDigitalOceanKubernetesConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanKubernetesClusterExists("digitalocean_kubernetes_cluster.foobar", &k8s),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "name", rName),
				),
			},
			{
				Config: testAccDigitalOceanKubernetesConfigBasic4(rName + "-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanKubernetesClusterExists("digitalocean_kubernetes_cluster.foobar", &k8s),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "tags.#", "2"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "tags.2053932785", "one"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "tags.298486374", "two"),
				),
			},
		},
	})
}

func TestAccDigitalOceanKubernetesCluster_UpdatePoolDetails(t *testing.T) {
	t.Parallel()
	rName := randomTestName()
	var k8s godo.KubernetesCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanKubernetesClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDigitalOceanKubernetesConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanKubernetesClusterExists("digitalocean_kubernetes_cluster.foobar", &k8s),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "name", rName),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.#", "1"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.name", "default"),
				),
			},
			{
				Config: testAccDigitalOceanKubernetesConfigBasic2(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanKubernetesClusterExists("digitalocean_kubernetes_cluster.foobar", &k8s),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "name", rName),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.#", "1"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.name", "default-rename"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.node_count", "2"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.tags.#", "3"),
				),
			},
		},
	})
}

func TestAccDigitalOceanKubernetesCluster_UpdatePoolSize(t *testing.T) {
	t.Parallel()
	rName := randomTestName()
	var k8s godo.KubernetesCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanKubernetesClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDigitalOceanKubernetesConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanKubernetesClusterExists("digitalocean_kubernetes_cluster.foobar", &k8s),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "name", rName),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.#", "1"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.size", "s-1vcpu-2gb"),
				),
			},
			{
				Config: testAccDigitalOceanKubernetesConfigBasic3(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanKubernetesClusterExists("digitalocean_kubernetes_cluster.foobar", &k8s),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "name", rName),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.#", "1"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.node_count", "1"),
					resource.TestCheckResourceAttr("digitalocean_kubernetes_cluster.foobar", "node_pool.0.size", "s-2vcpu-4gb"),
				),
			},
		},
	})
}

func TestAccDigitalOceanKubernetesCluster_KubernetesProviderInteroperability(t *testing.T) {
	t.Parallel()
	rName := randomTestName()
	var k8s godo.KubernetesCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanKubernetesClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDigitalOceanKubernetesConfig_KubernetesProviderInteroperability(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanKubernetesClusterExists("digitalocean_kubernetes_cluster.foobar", &k8s), resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "kube_config.0.raw_config"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "kube_config.0.cluster_ca_certificate"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "kube_config.0.host"),
					resource.TestCheckResourceAttrSet("digitalocean_kubernetes_cluster.foobar", "kube_config.0.token"),
				),
			},
		},
	})
}

func testAccDigitalOceanKubernetesConfigBasic(rName string) string {
	return fmt.Sprintf(`
resource "digitalocean_kubernetes_cluster" "foobar" {
	name    = "%s"
	region  = "lon1"
	version = "1.15.4-do.0"
	tags    = ["foo","bar", "one"]

	node_pool {
	  name = "default"
		size  = "s-1vcpu-2gb"
		node_count = 1
		tags  = ["one","two"]
	}
}
`, rName)
}

func testAccDigitalOceanKubernetesConfigBasic2(rName string) string {
	return fmt.Sprintf(`
resource "digitalocean_kubernetes_cluster" "foobar" {
	name    = "%s"
	region  = "lon1"
	version = "1.15.4-do.0"
	tags    = ["foo","bar"]

	node_pool {
	  name = "default-rename"
		size  = "s-1vcpu-2gb"
		node_count = 2
		tags  = ["one","two","three"]
	}
}
`, rName)
}

func testAccDigitalOceanKubernetesConfigBasic3(rName string) string {
	return fmt.Sprintf(`
resource "digitalocean_kubernetes_cluster" "foobar" {
	name    = "%s"
	region  = "lon1"
	version = "1.15.4-do.0"
	tags    = ["foo","bar"]

	node_pool {
	  name = "default"
		size  = "s-2vcpu-4gb"
		node_count = 1
		tags  = ["one","two"]
	}
}
`, rName)
}

func testAccDigitalOceanKubernetesConfigBasic4(rName string) string {
	return fmt.Sprintf(`
resource "digitalocean_kubernetes_cluster" "foobar" {
	name    = "%s"
	region  = "lon1"
	version = "1.15.4-do.0"
	tags    = ["one","two"]

	node_pool {
	  name = "default"
		size  = "s-2vcpu-4gb"
		node_count = 1
		tags  = ["foo","bar"]
	}
}
`, rName)
}

func testAccDigitalOceanKubernetesConfig_KubernetesProviderInteroperability(rName string) string {
	return fmt.Sprintf(`
resource "digitalocean_kubernetes_cluster" "foobar" {
	name    = "%s"
	region  = "lon1"
	version = "1.15.4-do.0"

	node_pool {
	  name = "default"
		size  = "s-2vcpu-4gb"
		node_count = 1
	}
}

provider "kubernetes" {
  host = digitalocean_kubernetes_cluster.foobar.endpoint

  client_certificate = base64decode(
    digitalocean_kubernetes_cluster.foobar.kube_config[0].client_certificate
  )
  client_key = base64decode(
    digitalocean_kubernetes_cluster.foobar.kube_config[0].client_key
  )
  cluster_ca_certificate = base64decode(
    digitalocean_kubernetes_cluster.foobar.kube_config[0].cluster_ca_certificate
  )
  token = digitalocean_kubernetes_cluster.foobar.kube_config[0].token
}

resource "kubernetes_service_account" "tiller" {
  metadata {
    name      = "tiller"
    namespace = "kube-system"
  }

  automount_service_account_token = true
}
`, rName)
}

func testAccCheckDigitalOceanKubernetesClusterDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*CombinedConfig).godoClient()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_kubernetes_cluster" {
			continue
		}

		// Try to find the cluster
		_, _, err := client.Kubernetes.Get(context.Background(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("K8s Cluster still exists")
		}
	}

	return nil
}

func testAccCheckDigitalOceanKubernetesClusterExists(n string, cluster *godo.KubernetesCluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*CombinedConfig).godoClient()

		foundCluster, _, err := client.Kubernetes.Get(context.Background(), rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundCluster.ID != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*cluster = *foundCluster

		return nil
	}
}

func Test_filterTags(t *testing.T) {
	tests := []struct {
		have []string
		want []string
	}{
		{
			have: []string{"k8s", "foo"},
			want: []string{"foo"},
		},
		{
			have: []string{"k8s", "k8s:looks-like-a-uuid", "bar"},
			want: []string{"bar"},
		},
		{
			have: []string{"k8s", "k8s:looks-like-a-uuid", "bar", "k8s-this-is-ok"},
			want: []string{"bar", "k8s-this-is-ok"},
		},
		{
			have: []string{"k8s", "k8s:looks-like-a-uuid", "terraform:default-node-pool", "baz"},
			want: []string{"baz"},
		},
	}

	for _, tt := range tests {
		filteredTags := filterTags(tt.have)
		if !reflect.DeepEqual(filteredTags, tt.want) {
			t.Errorf("filterTags returned %+v, expected %+v", filteredTags, tt.want)
		}
	}
}

func Test_renderKubeconfig(t *testing.T) {
	certAuth := []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURKekNDQWWlOQT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K")
	expected := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: %v
    server: https://6a37a0f6-c355-4527-b54d-521beffd9817.k8s.ondigitalocean.com
  name: do-lon1-test-cluster
contexts:
- context:
    cluster: do-lon1-test-cluster
    user: do-lon1-test-cluster-admin
  name: do-lon1-test-cluster
current-context: do-lon1-test-cluster
users:
- name: do-lon1-test-cluster-admin
  user:
    token: 97ae2bbcfd85c34155a56b822ffa73909d6770b28eb7e5dfa78fa83e02ffc60f
`, base64.StdEncoding.EncodeToString(certAuth))

	creds := godo.KubernetesClusterCredentials{
		Server:                   "https://6a37a0f6-c355-4527-b54d-521beffd9817.k8s.ondigitalocean.com",
		CertificateAuthorityData: certAuth,
		Token:                    "97ae2bbcfd85c34155a56b822ffa73909d6770b28eb7e5dfa78fa83e02ffc60f",
		ExpiresAt:                time.Now(),
	}
	kubeConfigRenderd, err := renderKubeconfig("test-cluster", "lon1", &creds)
	if err != nil {
		t.Errorf("error calling renderKubeconfig: %s", err)

	}
	got := string(kubeConfigRenderd)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("renderKubeconfig returned %+v\n, expected %+v\n", got, expected)
	}
}
