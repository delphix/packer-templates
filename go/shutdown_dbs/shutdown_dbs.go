package main

import (
	"fmt"

	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

// Options for the script
type Options struct {
	DDPName      string               `short:"e" long:"ddp_hostname" env:"DELPHIX_DDP_HOSTNAME" description:"The hostname or IP address of the Delphix Dynamic Data Platform" required:"true"`
	UserName     string               `short:"u" long:"username" env:"DELPHIX_USER" description:"The username used to authenticate to the Delphix Engine" required:"true"`
	Password     string               `short:"p" long:"password" env:"DELPHIX_PASS" description:"The password used to authenticate to the Delphix Engine" required:"true"`
	Debug        []bool               `short:"v" long:"debug" env:"DELPHIX_DEBUG" description:"Turn on debugging. -vvv for the most verbose debugging"`
	SkipValidate bool                 `long:"skip-validate" env:"DELPHIX_SKIP_VALIDATE" description:"Don't validate TLS certificate of Delphix Engine"`
	ConfigFile   func(s string) error `short:"c" long:"config" description:"Optional INI config file to pass in for the variables" no-ini:"true"`
	VDBList      []string             `long:"vdb" env:"DELPHIX_VDB" description:"The name of the VDB to enable and start" required:"true"`
}

func (c *Client) findSourceByDatabaseRef(databaseRef string) map[string]interface{} {
	namespace := "source"
	//Find our VDB of interest
	log.Info("Searching for source by reference")
	obj := c.listObjects(namespace, fmt.Sprintf("database=%s", databaseRef))
	log.Debug(obj)
	if obj == nil {
		log.Fatalf("Could not find sourceconfig for VDB reference %s", databaseRef)
	} else if len(obj) > 1 {
		log.Fatalf("More than one result was returned. Exiting.\n%v", obj)
	}
	return obj[0].(map[string]interface{})
}

func (c *Client) stopVDBByName(vdbName string, wait bool) (results map[string]interface{}) {
	namespace := "database"
	//Find our VDB of interest
	log.Info("Searching for VDB by name")
	obj := c.findObjectByName(namespace, vdbName)
	log.Debug(obj)
	if obj == nil {
		log.Fatalf("Could not find VDB named %s", vdbName)
	}
	log.Infof("Found %s: %s", obj["name"], obj["reference"])
	sourceObj := c.findSourceByDatabaseRef(obj["reference"].(string))
	//start
	namespace = "source"
	url := fmt.Sprintf("%s/%s/stop", namespace, sourceObj["reference"])
	action := c.httpPost(url, "")

	if wait {
		c.jobWaiter(action)
	}
	return action
}

func (c *Client) batchStopVDBByName(vdbList ...string) (resultsList []map[string]interface{}) {
	for _, v := range vdbList {
		action := c.stopVDBByName(v, false)
		resultsList = append(resultsList, action)
	}

	c.jobWaiter(resultsList...)

	return resultsList
}

var (
	opts             Options
	parser           = flags.NewParser(&opts, flags.Default)
	apiVersionString = "1.9.3"
	logger           *log.Entry
	url              string
	version          = "1.0.3"
)

func main() {
	var err error

	log.Info("Establishing session and logging in")
	client := NewClient(opts.UserName, opts.Password, fmt.Sprintf("https://%s/resources/json/delphix", opts.DDPName))
	client.initResty()
	err = client.waitForEngineReady(10, 600)

	if err != nil {
		log.Fatal(err)
	}
	log.Info("Successfully Logged in")

	client.batchStopVDBByName(opts.VDBList...)
	log.Info("Complete")
}
