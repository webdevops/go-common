package collector

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/webdevops/go-common/azuresdk/armclient"
	"github.com/webdevops/go-common/utils/to"
)

type (
	cacheSpecDef struct {
		protocol string
		url      *url.URL

		tag *string

		raw string

		spec map[string]string

		client interface{}
	}
)

const (
	cacheProtocolFile         = "file"
	cacheProtocolAzBlob       = "azblob"
	cacheProtocolK8sConfigMap = "k8scm"
)

// BuildCacheTag builds a cache tag based on prefix string and various interfaces, returns a tag value (string)
func BuildCacheTag(prefix string, val ...interface{}) *string {
	ret := prefix

	if len(val) > 0 {
		tagPayload, err := json.Marshal(val)
		if err != nil {
			panic(err)
		}

		hasher := sha256.New()
		hasher.Write(tagPayload)
		ret += "." + base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	}

	return &ret
}

// EnableCache alias of SetCache
func (c *Collector) EnableCache(cache string, cacheTag *string) {
	c.SetCache(&cache, cacheTag)
}

// SetCache enables caching of collector with local file and azblob support
//
//	  cache can be specified as local file or storageaccount blob:
//	    path or file://path/to/file will store cached metrics in file
//		   azblob://storageaccount.blob.core.windows.net/container/blob will store cached metrics in storageaccount
//		 cacheTag is used to force restore, if nil cacheTag is ignored and otherwise enforced
func (c *Collector) SetCache(cache *string, cacheTag *string) {
	if cache == nil {
		c.cache = nil
		return
	}

	rawSpec := *cache

	c.cache = &cacheSpecDef{
		raw:  rawSpec,
		spec: map[string]string{},
		tag:  cacheTag,
	}

	switch {
	case strings.HasPrefix(rawSpec, `file://`):
		c.cache.protocol = cacheProtocolFile
		c.cache.spec["file:path"] = strings.TrimPrefix(rawSpec, "file://")
	case strings.HasPrefix(rawSpec, `azblob://`):
		c.cache.protocol = cacheProtocolAzBlob
		parsedUrl, err := url.Parse(rawSpec)
		if err != nil {
			c.logger.Fatal(err)
		}
		c.cache.url = parsedUrl

		azureClient, err := armclient.NewArmClientFromEnvironment(c.logger)
		if err != nil {
			c.logger.Fatal(err)
		}

		storageAccount := fmt.Sprintf(`https://%v/`, c.cache.url.Hostname())
		pathParts := strings.Split(c.cache.url.Path, "/")
		if len(pathParts) < 2 {
			c.logger.Fatalf(`azblob path needs to be specified as azblob://storageaccount.blob.core.windows.net/container/blob, got: %v`, rawSpec)
		}

		c.cache.spec["azblob:container"] = pathParts[0]
		c.cache.spec["azblob:blob"] = strings.Join(pathParts[1:], "/")

		// create a client for the specified storage account
		azblobOpts := azblob.ClientOptions{ClientOptions: *azureClient.NewAzCoreClientOptions()}
		client, err := azblob.NewClient(storageAccount, azureClient.GetCred(), &azblobOpts)
		if err != nil {
			c.logger.Fatal(err)
		}

		c.cache.client = client

	case strings.HasPrefix(rawSpec, `k8scm://`):
		c.cache.protocol = cacheProtocolK8sConfigMap
		parsedUrl, err := url.Parse(rawSpec)
		if err != nil {
			c.logger.Fatal(err)
		}
		c.cache.url = parsedUrl
		pathParts := strings.SplitN(parsedUrl.Path, "/", 3)
		if len(pathParts) < 3 {
			c.logger.Fatalf(`azblob path needs to be specified as k8scm://namespace/name, got: %v`, rawSpec)
		}

		c.cache.spec["kubernetes:namespace"] = c.cache.url.Hostname()
		// pathParts[0] is always empty, since the .Path begins with an /
		c.cache.spec["kubernetes:configmap"] = pathParts[1]
		// Slashes are not allowed as key
		c.cache.spec["kubernetes:configmapKey"] = strings.ReplaceAll(pathParts[2], "/", "-")

		// creates the in-cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			c.logger.Fatal(err)
		}
		// creates the client
		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			c.logger.Fatal(err.Error())
		}

		c.cache.client = client.CoreV1()
	default:
		c.cache.protocol = cacheProtocolFile
		c.cache.spec["file:path"] = rawSpec
	}
}

// DisableCache disables all caching
func (c *Collector) DisableCache() {
	c.cache = nil
}

// collectionRestoreCache tries to restore metrics from cache
func (c *Collector) collectionRestoreCache() bool {
	if c.cache == nil {
		return false
	}

	if cacheContent, exists := c.cacheRead(); exists {
		restoredData := NewCollectorData()

		c.logger.Infof(`restoring state from cache: %s`, c.cache.raw)

		err := json.Unmarshal(cacheContent, &restoredData)
		if err == nil {
			if c.cache.tag != nil {
				if restoredData.Tag == nil || to.String(c.cache.tag) != to.String(restoredData.Tag) {
					// cache tag check is enforced but there is a mismatch
					c.logger.Infof(`cache tag mismatch, ignoring cache`)
					return false
				}
			}

			if restoredData.Expiry != nil && restoredData.Expiry.After(time.Now()) {
				// restore data
				c.data.Expiry = restoredData.Expiry
				for name, restoreMetricList := range restoredData.Metrics {
					if restoreMetricList.List == nil {
						continue
					}

					if metricList, exists := c.data.Metrics[name]; exists {
						metricList.List = restoreMetricList.List
						metricList.Init()
					}
				}

				// calculate sleep time for next collect run
				// but sleep time should not exceed defined scrape time
				sleepTime := time.Until(*c.data.Expiry) + 1*time.Minute
				if c.scrapeTime != nil && sleepTime < *c.scrapeTime {
					c.SetNextSleepDuration(sleepTime)
				}

				// restore last scrape time from cache
				if restoredData.Created != nil {
					c.lastScrapeTime = restoredData.Created
				}

				c.logger.Infof(`restored state from cache: "%s" (expiring %s)`, c.cache.raw, c.data.Expiry.UTC().String())
				return true
			} else {
				c.logger.Infof(`ignoring cached state, already expired`)
			}
		} else {
			c.logger.Warnf(`unable to decode cache: %v`, err.Error())
		}
	}

	return false
}

// collectionSaveCache saves current metrics to cache
func (c *Collector) collectionSaveCache() {
	if c.cache == nil {
		return
	}

	expiryTime := time.Now().Add(*c.sleepTime)
	c.data.Created = &c.collectionStartTime
	c.data.Expiry = &expiryTime
	c.data.Tag = c.cache.tag

	if jsonData, err := json.Marshal(c.data); err == nil {
		c.cacheStore(jsonData)
		c.logger.Infof(`saved state to cache: %s (expiring %s)`, c.cache.raw, c.data.Expiry.UTC().String())
	} else {
		c.logger.Errorf(`failed to serialize state for cache: %v`, err.Error())
	}

}

// cacheRead reads content from cache
func (c *Collector) cacheRead() ([]byte, bool) {
	switch c.cache.protocol {
	case cacheProtocolFile:
		filePath := c.cache.spec["file:path"]
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			content, _ := os.ReadFile(filePath) // #nosec inside container
			return content, true
		}
	case cacheProtocolAzBlob:
		response, err := c.cache.client.(*azblob.Client).DownloadStream(c.context, c.cache.spec["azblob:container"], c.cache.spec["azblob:blob"], nil)
		if err == nil {
			if content, err := io.ReadAll(response.Body); err == nil {
				return content, true
			}
		}
	case cacheProtocolK8sConfigMap:
		configMap, err := c.cache.client.(*corev1.CoreV1Client).ConfigMaps(c.cache.spec["kubernetes:namespace"]).Get(context.TODO(), c.cache.spec["kubernetes:configmap"], metav1.GetOptions{})

		if err == nil {
			if response, ok := configMap.BinaryData[c.cache.spec["kubernetes:configmapKey"]]; ok {
				r, err := gzip.NewReader(base64.NewDecoder(base64.StdEncoding, bytes.NewReader(response)))
				if err == nil {
					content, err := io.ReadAll(r)
					if err == nil {
						return content, true
					}
				}
			}
		}
	}

	return nil, false
}

// cacheRead saves content to cache
func (c *Collector) cacheStore(content []byte) {
	switch c.cache.protocol {
	case cacheProtocolFile:
		filePath := c.cache.spec["file:path"]

		dirPath := filepath.Dir(filePath)

		// ensure directory
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			err := os.Mkdir(dirPath, 0700)
			if err != nil {
				c.logger.Fatal(err)
			}
		}

		// calc tmp filename
		tmpFilePath := filepath.Join(
			dirPath,
			fmt.Sprintf(
				".%s.tmp",
				filepath.Base(filePath),
			),
		)

		// write to temp file first
		err := os.WriteFile(tmpFilePath, content, 0600) // #nosec inside container
		if err != nil {
			c.logger.Fatal(err)
		}

		// rename file to final cache file (atomic operation)
		err = os.Rename(tmpFilePath, filePath)
		if err != nil {
			c.logger.Fatal(err)
		}
	case cacheProtocolAzBlob:
		_, err := c.cache.client.(*azblob.Client).UploadBuffer(c.context, c.cache.spec["azblob:container"], c.cache.spec["azblob:blob"], content, nil)
		if err != nil {
			c.logger.Fatal(err)
		}
	case cacheProtocolK8sConfigMap:
		// Since the kubernetes configmap can only hold 1MB of data in total, we compress the data before store them
		var buf64 bytes.Buffer
		wb64 := base64.NewEncoder(base64.StdEncoding, &buf64)
		wgz := gzip.NewWriter(wb64)
		if _, err := wgz.Write(content); err != nil {
			c.logger.Fatal(err)
		}
		if err := wgz.Close(); err != nil {
			c.logger.Fatal(err)
		}
		if err := wb64.Close(); err != nil {
			c.logger.Fatal(err)
		}

		configMap := corev1apply.ConfigMap(c.cache.spec["kubernetes:configmap"], c.cache.spec["kubernetes:namespace"])
		configMap.WithBinaryData(map[string][]byte{c.cache.spec["kubernetes:configmapKey"]: buf64.Bytes()})

		_, err := c.cache.client.(*corev1.CoreV1Client).ConfigMaps(c.cache.spec["kubernetes:namespace"]).Apply(
			context.TODO(),
			configMap,
			metav1.ApplyOptions{
				Force:        false,
				FieldManager: "webdevops/common/" + c.cache.spec["kubernetes:configmapKey"],
			},
		)

		if err != nil {
			c.logger.Fatalf("Unable to update kubernetes configmap: %v", err)
		}
	}
}
